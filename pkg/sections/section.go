// Package sections models the individual multipart sections of an MHT file:
// their headers, decoded bodies, and re-encoding on output.
package sections

import (
	"io"
	"maps"
	"net/textproto"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/pkg/bodies"
	"github.com/goodblaster/go-mht/pkg/headers"
)

// Section is a single multipart section. Body holds the content, without
// the line break that precedes the next boundary: decoded text for
// quoted-printable sections, the undecoded base64 string (without line
// breaks) for base64 sections, and the bytes as is for identity encodings
// (7bit, 8bit, binary).
type Section struct {
	Headers textproto.MIMEHeader `json:"headers"`
	Body    string               `json:"content"`

	// layout remembers how the headers were written in the source so that
	// untouched headers are written back byte for byte.
	layout *headers.Layout
}

// New returns an empty Section with initialized headers.
func New() *Section {
	return &Section{
		Headers: make(textproto.MIMEHeader),
	}
}

// Write writes the section (headers, blank line, encoded body) to writer.
// The opening boundary is written by the caller. If the section has no
// Content-Transfer-Encoding header, the output declares quoted-printable —
// some ingestion paths strip the header. The section itself is not modified.
func (section *Section) Write(writer io.Writer) error {
	header := section.Headers

	if header.Get("Content-Transfer-Encoding") == "" {
		header = maps.Clone(header)
		if header == nil {
			header = make(textproto.MIMEHeader)
		}
		header.Set("Content-Transfer-Encoding", string(bodies.QuotedPrintable))
	}

	if err := headers.WriteAll(writer, header, section.layout); err != nil {
		return errors.Wrap(err, "failed to write headers")
	}

	// Blank line separating headers from body.
	if _, err := io.WriteString(writer, "\r\n"); err != nil {
		return errors.Wrap(err, "failed to write empty line")
	}

	switch {
	case section.IsBase64():
		if _, err := writeBase64(writer, section.Body); err != nil {
			return errors.Wrap(err, "failed to write base64 body")
		}
	case section.IsQuotedPrintable():
		if _, err := writeQuotedPrintable(writer, section.Body); err != nil {
			return errors.Wrap(err, "failed to write quoted-printable body")
		}
	default:
		// Identity encodings (7bit, 8bit, binary): body is written verbatim,
		// followed by the line break that precedes the next boundary.
		if _, err := io.WriteString(writer, section.Body+"\r\n"); err != nil {
			return errors.Wrap(err, "failed to write raw body")
		}
	}

	return nil
}

// encoding returns the section's normalized Content-Transfer-Encoding.
func (section *Section) encoding() bodies.TransferEncoding {
	return bodies.ParseTransferEncoding(section.Headers.Get("Content-Transfer-Encoding"))
}

// IsBase64 reports whether the section body is base64 encoded.
func (section *Section) IsBase64() bool {
	return section.encoding() == bodies.Base64
}

// IsQuotedPrintable reports whether the section body is quoted-printable
// encoded. A missing Content-Transfer-Encoding counts as quoted-printable,
// since some ingestion paths strip the header.
func (section *Section) IsQuotedPrintable() bool {
	encoding := section.encoding()
	return encoding == bodies.QuotedPrintable || encoding == ""
}

// writeBase64 writes body wrapped at 76 characters per line, matching Blink
// output, followed by the blank line that precedes the next boundary.
func writeBase64(writer io.Writer, body string) (int, error) {
	var n int
	for i := 0; i < len(body); i += 76 {
		end := min(i+76, len(body))

		written, err := io.WriteString(writer, body[i:end]+"\r\n")
		n += written
		if err != nil {
			return n, errors.Wrap(err, "failed to write base64 line")
		}
	}

	written, err := io.WriteString(writer, "\r\n")
	n += written
	if err != nil {
		return n, errors.Wrap(err, "failed to write blank line")
	}

	return n, nil
}

// QuotedPrintableEncode encodes input in quoted-printable format, following
// Blink's conventions (see bodies.EncodeQuotedPrintable).
func QuotedPrintableEncode(input string) (string, error) {
	return bodies.EncodeQuotedPrintable(input), nil
}

// writeQuotedPrintable writes body quoted-printable encoded, followed by the
// line break that precedes the next boundary.
func writeQuotedPrintable(writer io.Writer, body string) (int, error) {
	n, err := io.WriteString(writer, bodies.EncodeQuotedPrintable(body)+"\r\n")
	if err != nil {
		return n, errors.Wrap(err, "failed to write quoted-printable")
	}

	return n, nil
}
