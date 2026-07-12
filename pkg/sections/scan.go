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
	if err := headers.Scan(scanner, &section.Headers); err != nil {
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		return errors.Wrap(err, "failed to scan section headers")
	}

	encoding := bodies.TransferEncoding(section.Headers.Get("Content-Transfer-Encoding"))

	var body strings.Builder
	err := bodies.Scan(scanner, boundary, encoding, &body)
	section.Body = body.String()

	if err != nil {
		if errors.Is(err, io.EOF) {
			return io.EOF
		}
		return errors.Wrap(err, "failed to scan section body")
	}

	return nil
}
