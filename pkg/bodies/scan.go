package bodies

import (
	"bufio"
	"io"
	"strings"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/internal/lines"
)

// Scan reads body lines from scanner into body until it reaches the next
// boundary.
//
//   - Base64 lines are concatenated without line breaks.
//   - Quoted-printable lines are decoded.
//   - A missing encoding is treated as quoted-printable, matching Blink
//     output where the header has been stripped. If the content turns out
//     not to be valid quoted-printable it is kept verbatim instead, so a
//     stray "=" in an undeclared part does not fail the parse.
//   - Any other encoding (7bit, 8bit, binary) is passed through unchanged.
//     When scanner preserves line terminators (see documents.Parse) they
//     are kept exactly; otherwise each line is terminated with CRLF.
//
// Scan returns nil when it stops at an intermediate boundary and io.EOF when
// it stops at the closing boundary or the end of input.
func Scan(scanner *bufio.Scanner, boundary string, encoding TransferEncoding, body *strings.Builder) error {
	encoding = ParseTransferEncoding(string(encoding))

	// For an undeclared encoding, keep the verbatim text alongside the
	// decoded text until we know the content really is quoted-printable.
	var verbatim strings.Builder
	undeclared := encoding == ""

	for scanner.Scan() {
		raw := scanner.Text()
		line := lines.Trim(raw)
		if line == boundary {
			return nil
		}
		if line == boundary+"--" {
			return io.EOF
		}

		switch encoding {
		case Base64:
			body.WriteString(line)
		case QuotedPrintable, "":
			decoded, err := DecodeQuotedPrintableLine(line)
			if err != nil {
				if !undeclared {
					return errors.Wrap(err, "failed to decode quoted-printable line")
				}
				// Not quoted-printable after all: fall back to the
				// verbatim text for the whole part.
				body.Reset()
				body.WriteString(verbatim.String())
				verbatim.Reset()
				encoding = identity
				writeVerbatim(body, raw, line)
				continue
			}
			body.WriteString(decoded)
			if undeclared {
				writeVerbatim(&verbatim, raw, line)
			}
		default:
			writeVerbatim(body, raw, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to scan body")
	}

	return io.EOF
}

// identity stands for any encoding whose content is stored as is.
const identity TransferEncoding = "identity"

// writeVerbatim appends a line unchanged. raw is the line as scanned and
// line is the same text without its terminator; when the scanner has already
// stripped the terminator, CRLF is assumed.
func writeVerbatim(out *strings.Builder, raw, line string) {
	if raw == line {
		out.WriteString(line)
		out.WriteString("\r\n")
		return
	}
	out.WriteString(raw)
}
