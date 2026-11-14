package headers

import (
	"io"
	"strings"

	"github.com/goodblaster/errors"
)

// Write writes one header (all of its values) to writer. Content-Type values
// are folded at their parameters the way Blink does; every other header is
// written on a single line, since a semicolon in a page title or a URL is
// not a place to fold.
func Write(writer io.Writer, key string, values []string) error {
	for _, value := range values {
		if strings.EqualFold(key, "Content-Type") {
			value = fold(value)
		}
		if _, err := io.WriteString(writer, key+": "+value+"\r\n"); err != nil {
			return errors.Wrap(err, "failed to write header")
		}
	}
	return nil
}

// fold re-folds a header value at semicolons, ignoring semicolons inside
// double-quoted strings (e.g. a boundary parameter).
func fold(value string) string {
	parts := splitOutsideQuotes(value, ';')
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return strings.Join(parts, ";\r\n\t")
}

// splitOutsideQuotes splits s on sep, treating double-quoted runs as opaque.
func splitOutsideQuotes(s string, sep byte) []string {
	var parts []string
	var start int
	inQuotes := false
	for i := range len(s) {
		switch s[i] {
		case '"':
			inQuotes = !inQuotes
		case sep:
			if !inQuotes {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}
