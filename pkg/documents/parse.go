package documents

import (
	"bufio"
	"fmt"
	"io"
	"mime"
	"os"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/internal/lines"
	"github.com/goodblaster/go-mht/pkg/headers"
	"github.com/goodblaster/go-mht/pkg/sections"
)

// maxLineSize is the largest single line Parse will accept. MHT bodies are
// normally wrapped well under 76 characters, but identity-encoded parts can
// contain arbitrarily long lines (e.g. minified scripts).
const maxLineSize = 16 * 1024 * 1024

// Parse reads an MHT document from reader.
func Parse(reader io.Reader) (*Document, error) {
	doc := New()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	// Keep line terminators so that binary parts survive byte for byte.
	scanner.Split(lines.Split)

	// Top-level document headers.
	layout, err := headers.ScanLayout(scanner, &doc.Headers)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan document headers")
	}
	doc.layout = layout

	_, params, err := mime.ParseMediaType(doc.Headers.Get("Content-Type"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse Content-Type")
	}

	boundary := params["boundary"]
	doc.Boundary = boundary

	if boundary == "" {
		return nil, errors.New("boundary not found in Content-Type header")
	}

	// Boundary lines carry two leading hyphens.
	boundary = "--" + boundary

	// Skip the preamble up to the first boundary.
	if err := doc.FindBoundary(scanner, boundary); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	// Read all sections.
	for {
		section := sections.New()
		err := sections.Scan(scanner, boundary, section)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		// A boundary followed by nothing (a file cut off right after one,
		// or a missing closing boundary) is not a section.
		if len(section.Headers) > 0 || section.Body != "" {
			doc.Sections = append(doc.Sections, *section)
		}

		if errors.Is(err, io.EOF) {
			doc.Reindex()
			return doc, nil
		}
	}
}

// ParseFile reads an MHT document from the named file.
func ParseFile(filename string) (*Document, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	return Parse(file)
}

// FindBoundary advances scanner to the next boundary line. It returns nil at
// an intermediate boundary and io.EOF at the closing boundary or end of
// input.
func (doc *Document) FindBoundary(scanner *bufio.Scanner, boundary string) error {
	for scanner.Scan() {
		line := lines.Trim(scanner.Text())
		if line == boundary {
			return nil
		}
		if line == boundary+"--" {
			return io.EOF
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to find boundary")
	}

	return io.EOF
}

// Write writes the document to writer in Blink's MHTML format. For a
// document produced by Parse and not modified, the output is byte-identical
// to the input.
func (doc *Document) Write(writer io.Writer) error {
	// Document headers.
	if err := headers.WriteAll(writer, doc.Headers, doc.layout); err != nil {
		return errors.Wrap(err, "failed to write headers")
	}

	// Header terminator plus the blank line before the first boundary.
	if _, err := io.WriteString(writer, "\r\n\r\n"); err != nil {
		return errors.Wrap(err, "failed to write empty lines")
	}

	// Sections.
	for i := range doc.Sections {
		if err := doc.StartSection(writer); err != nil {
			return err
		}
		if err := doc.Sections[i].Write(writer); err != nil {
			return errors.Wrap(err, "failed to write section")
		}
	}

	// Closing boundary.
	return doc.EndSection(writer)
}

// StartSection writes an intermediate boundary line.
func (doc *Document) StartSection(writer io.Writer) error {
	if _, err := fmt.Fprintf(writer, "--%s\r\n", doc.Boundary); err != nil {
		return errors.Wrap(err, "failed to write section start")
	}
	return nil
}

// EndSection writes the closing boundary line.
func (doc *Document) EndSection(writer io.Writer) error {
	if _, err := fmt.Fprintf(writer, "--%s--\r\n", doc.Boundary); err != nil {
		return errors.Wrap(err, "failed to write section end")
	}
	return nil
}
