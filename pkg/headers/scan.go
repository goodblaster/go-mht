// Package headers reads and writes MIME headers while preserving the header
// order and folding style used by Blink-generated MHTML files.
package headers

import (
	"bufio"
	"io"
	"net/textproto"
	"strings"

	"github.com/goodblaster/errors"
)

// Scan reads a header block from scanner into headers. It stops at the first
// blank line after at least one header has been read (blank lines before the
// block are tolerated). Folded continuation lines (starting with a tab or
// space) are unfolded into the preceding header value.
//
// Scan returns nil when the block ends normally and io.EOF when the input
// ends before a terminating blank line.
func Scan(scanner *bufio.Scanner, headers *textproto.MIMEHeader) error {
	var lastKey string
	for scanner.Scan() {
		line := scanner.Text()

		// Blank lines above the block are tolerated; the first blank line
		// after it terminates the block.
		if line == "" {
			if len(*headers) == 0 {
				continue
			}
			return nil
		}

		// Folded continuation line.
		if (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, " ")) && lastKey != "" {
			key := textproto.CanonicalMIMEHeaderKey(lastKey)
			values := (*headers)[key]
			values[len(values)-1] += " " + strings.TrimSpace(line)
			continue
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			return errors.Newf("malformed header line: %q", line)
		}
		key = strings.TrimSpace(key)
		headers.Add(key, strings.TrimSpace(value))
		lastKey = key
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "failed to scan headers")
	}

	return io.EOF
}
