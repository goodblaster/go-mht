package bodies

import (
	"bufio"
	"io"
	"strings"

	"github.com/goodblaster/errors"
)

// Scan reads body lines from scanner into body until it reaches the next
// boundary. Base64 lines are concatenated without line breaks,
// quoted-printable lines are decoded (a missing encoding is treated as
// quoted-printable, matching Blink output where the header is sometimes
// stripped), and any other encoding is passed through unchanged.
//
// Scan returns nil when it stops at an intermediate boundary and io.EOF when
// it stops at the closing boundary or the end of input.
func Scan(scanner *bufio.Scanner, boundary string, encoding TransferEncoding, body *strings.Builder) error {
	for scanner.Scan() {
		line := scanner.Text()
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
				return errors.Wrap(err, "failed to decode quoted-printable line")
			}
			body.WriteString(decoded)
		default:
			// Identity encodings (7bit, 8bit, binary): keep the line as is.
			body.WriteString(line)
			body.WriteString("\r\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to scan body")
	}

	return io.EOF
}
