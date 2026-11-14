package sections

import (
	"bufio"
	"io"
	"strings"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/pkg/bodies"
	"github.com/goodblaster/go-mht/pkg/headers"
)

// Scan reads one section (headers and body) from scanner into section. It
// assumes the opening boundary has already been consumed. Scan returns nil
// when it stops at an intermediate boundary and io.EOF when it stops at the
// closing boundary or the end of input; in both cases the section is valid.
func Scan(scanner *bufio.Scanner, boundary string, section *Section) error {
	layout, err := headers.ScanLayout(scanner, &section.Headers)
	section.layout = layout
	if err != nil {
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		return errors.Wrap(err, "failed to scan section headers")
	}

	var body strings.Builder
	err = bodies.Scan(scanner, boundary, section.encoding(), &body)
	section.Body = body.String()

	// The line break before a boundary belongs to the boundary, not to the
	// content. Dropping it leaves Body holding exactly the content, which
	// Write then terminates again. (Base64 bodies hold no line breaks.)
	if !section.IsBase64() {
		if trimmed, found := strings.CutSuffix(section.Body, "\r\n"); found {
			section.Body = trimmed
		} else {
			section.Body = strings.TrimSuffix(section.Body, "\n")
		}
	}

	if err != nil {
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		return errors.Wrap(err, "failed to scan section body")
	}

	return nil
}
