package bodies

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/goodblaster/errors"
)

// DecodeQuotedPrintableLine decodes a single quoted-printable encoded line
// (without its trailing CRLF). A hard line break appends CRLF to the result;
// a soft line break (trailing "=") appends nothing, so consecutive decoded
// lines concatenate into the original text.
//
// The standard library's quotedprintable.Reader is not used here: it is
// stream-oriented and normalizes input in ways that break byte-identical
// round trips of Blink-generated MHTML.
func DecodeQuotedPrintableLine(input string) (string, error) {
	var buffer bytes.Buffer

	length := len(input)
	softBreak := false

	for i := 0; i < length; i++ {
		if input[i] != '=' {
			buffer.WriteByte(input[i])
			continue
		}

		// "=" at the end of the line is a soft line break.
		if i == length-1 {
			softBreak = true
			break
		}

		// Otherwise "=" must introduce a two-digit hex escape (e.g. =3D).
		if i+2 < length && isHexDigit(input[i+1]) && isHexDigit(input[i+2]) {
			hexValue := input[i+1 : i+3]
			decodedByte, err := strconv.ParseUint(hexValue, 16, 8)
			if err != nil {
				return "", errors.Wrapf(err, "invalid quoted-printable encoding: =%s", hexValue)
			}
			buffer.WriteByte(byte(decodedByte))
			i += 2
		} else {
			return "", errors.Newf("invalid quoted-printable sequence at byte %d", i)
		}
	}

	if softBreak {
		return buffer.String(), nil
	}

	return buffer.String() + "\r\n", nil
}

// isHexDigit reports whether c is a valid hexadecimal digit (0-9, A-F, a-f).
func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

// maxLineLength is the longest encoded line Blink emits, including the "="
// of a soft line break.
const maxLineLength = 76

// EncodeQuotedPrintable encodes input the way Blink does when it writes an
// MHTML part, so that re-encoding a decoded Blink body reproduces the
// original bytes. It differs from mime/quotedprintable in how it treats the
// final character of the input:
//
//   - a line ending that is the last character is escaped (=0A or =0D)
//     rather than written as a line break;
//   - the last character needs no room for a soft line break after it, so
//     the final line may be a full 76 columns instead of 75.
//
// Every other line ending (LF, CR, or CRLF) is written as CRLF. The result
// does not include the CRLF that separates the body from the next boundary.
func EncodeQuotedPrintable(input string) string {
	const hex = "0123456789ABCDEF"

	var out strings.Builder
	out.Grow(len(input) + len(input)/8)

	lineLength := 0
	for i := 0; i < len(input); i++ {
		c := input[i]
		last := i == len(input)-1

		// Non-printable ASCII and "=" are escaped; tabs and spaces only
		// at the end of a line.
		escape := (c < ' ' || c > '~' || c == '=') && c != '\t'
		if !escape && (c == '\t' || c == ' ') {
			escape = last || lineEndingLength(input, i+1) > 0
		}

		if !last {
			if n := lineEndingLength(input, i); n > 0 {
				out.WriteString("\r\n")
				lineLength = 0
				i += n - 1
				continue
			}
		}

		encodedLength := 1
		if escape {
			encodedLength = 3
		}
		if !last {
			// Leave room for the "=" of a soft line break.
			encodedLength++
		}
		if lineLength+encodedLength > maxLineLength {
			out.WriteString("=\r\n")
			lineLength = 0
		}

		if escape {
			out.WriteByte('=')
			out.WriteByte(hex[c>>4])
			out.WriteByte(hex[c&0x0F])
			lineLength += 3
		} else {
			out.WriteByte(c)
			lineLength++
		}
	}

	return out.String()
}

// lineEndingLength returns the length of the line ending starting at index
// (1 for a lone LF or CR, 2 for CRLF), or 0 if there is none.
func lineEndingLength(input string, index int) int {
	if index >= len(input) {
		return 0
	}
	switch input[index] {
	case '\n':
		return 1
	case '\r':
		if index+1 < len(input) && input[index+1] == '\n' {
			return 2
		}
		return 1
	}
	return 0
}
