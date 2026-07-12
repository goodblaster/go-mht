// Package sections models the individual multipart sections of an MHT file:
// their headers, decoded bodies, and re-encoding on output.
package sections

import (
	"bytes"
	"io"
	"mime/quotedprintable"
	"net/textproto"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/pkg/bodies"
	"github.com/goodblaster/go-mht/pkg/headers"
)

// Section is a single multipart section. Body holds the decoded content:
// text for quoted-printable sections, the undecoded base64 string (without
// line breaks) for base64 sections.
type Section struct {
	Headers textproto.MIMEHeader `json:"headers"`
	Body    string               `json:"content"`
}

// New returns an empty Section with initialized headers.
func New() *Section {
	return &Section{
		Headers: make(textproto.MIMEHeader),
	}
}

// Write writes the section (headers, blank line, encoded body) to writer.
// The opening boundary is written by the caller. If the section has no
// Content-Transfer-Encoding header, one is added, defaulting to
// quoted-printable — some ingestion paths strip it.
func (section *Section) Write(writer io.Writer) error {
	header := section.Headers

	if header.Get("Content-Transfer-Encoding") == "" {
		header.Set("Content-Transfer-Encoding", string(bodies.QuotedPrintable))
	}

	if err := headers.Ordered(header, func(key string, values []string) error {
		if err := headers.Write(writer, key, values); err != nil {
			return errors.Wrap(err, "failed to write headers")
		}
		return nil
	}); err != nil {
		return err
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
		// Identity encodings (7bit, 8bit, binary): body is written verbatim.
		if _, err := io.WriteString(writer, section.Body); err != nil {
			return errors.Wrap(err, "failed to write raw body")
		}
	}

	return nil
}

// IsBase64 reports whether the section body is base64 encoded.
func (section *Section) IsBase64() bool {
	return section.Headers.Get("Content-Transfer-Encoding") == "base64"
}

// IsQuotedPrintable reports whether the section body is quoted-printable
// encoded. A missing Content-Transfer-Encoding counts as quoted-printable,
// since some ingestion paths strip the header.
func (section *Section) IsQuotedPrintable() bool {
	encoding := section.Headers.Get("Content-Transfer-Encoding")
	return encoding == "quoted-printable" || encoding == ""
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

// QuotedPrintableEncode encodes input in quoted-printable format.
func QuotedPrintableEncode(input string) (string, error) {
	var buf bytes.Buffer
	writer := quotedprintable.NewWriter(&buf)
	if _, err := writer.Write([]byte(input)); err != nil {
		return "", errors.Wrap(err, "failed to write quoted-printable")
	}
	if err := writer.Close(); err != nil {
		return "", errors.Wrap(err, "failed to flush quoted-printable")
	}

	return buf.String(), nil
}

func writeQuotedPrintable(writer io.Writer, body string) (int, error) {
	out, err := QuotedPrintableEncode(body)
	if err != nil {
		return 0, errors.Wrap(err, "failed to encode quoted-printable")
	}

	n, err := io.WriteString(writer, out)
	if err != nil {
		return n, errors.Wrap(err, "failed to write quoted-printable")
	}

	return n, nil
}
