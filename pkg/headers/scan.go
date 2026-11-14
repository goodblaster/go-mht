// Package headers reads and writes MIME headers while preserving the header
// order and folding style used by Blink-generated MHTML files.
package headers

import (
	"bufio"
	"io"
	"net/textproto"
	"strings"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/internal/lines"
)

// Scan reads a header block from scanner into headers. It stops at the first
// blank line after at least one header has been read (blank lines before the
// block are tolerated). Folded continuation lines (starting with a tab or
// space) are unfolded into the preceding header value.
//
// Scan returns nil when the block ends normally and io.EOF when the input
// ends before a terminating blank line.
func Scan(scanner *bufio.Scanner, headers *textproto.MIMEHeader) error {
	_, err := ScanLayout(scanner, headers)
	return err
}

// ScanLayout is Scan, additionally returning a Layout that records how the
// block was written in the source. Passing that Layout to WriteAll
// reproduces the block byte for byte. The Layout is valid even when the
// returned error is io.EOF.
func ScanLayout(scanner *bufio.Scanner, headers *textproto.MIMEHeader) (*Layout, error) {
	layout := &Layout{}

	for scanner.Scan() {
		line := lines.Trim(scanner.Text())

		// Blank lines above the block are tolerated; the first blank line
		// after it terminates the block.
		if line == "" {
			if len(layout.fields) == 0 {
				continue
			}
			return layout, nil
		}

		// Folded continuation line.
		if (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ")) && len(layout.fields) > 0 {
			field := &layout.fields[len(layout.fields)-1]
			values := (*headers)[field.key]
			values[len(values)-1] += " " + strings.TrimSpace(line)
			field.value = values[len(values)-1]
			field.raw += line + "\r\n"
			continue
		}

		name, value, found := strings.Cut(line, ":")
		if !found {
			return layout, errors.Newf("malformed header line: %q", line)
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		headers.Add(name, value)
		layout.fields = append(layout.fields, layoutField{
			name:  name,
			key:   textproto.CanonicalMIMEHeaderKey(name),
			value: value,
			raw:   line + "\r\n",
		})
	}

	if err := scanner.Err(); err != nil {
		return layout, errors.Wrap(err, "failed to scan headers")
	}

	return layout, io.EOF
}
