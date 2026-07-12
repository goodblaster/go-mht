package bodies

import (
	"bytes"
	"strconv"

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
