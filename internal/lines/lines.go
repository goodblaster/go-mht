// Package lines splits input into lines without discarding their
// terminators, so that content which must survive byte for byte (binary
// MHT parts) can be reproduced exactly.
package lines

import "bytes"

// Split is a bufio.SplitFunc that returns each line including its
// terminator ("\r\n" or "\n"). The final line is returned as is when the
// input does not end with a terminator.
func Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i+1], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// Trim removes one trailing line terminator ("\r\n" or "\n") from line. A
// line with no terminator, as produced by bufio.ScanLines, is returned
// unchanged.
func Trim(line string) string {
	n := len(line)
	if n == 0 || line[n-1] != '\n' {
		return line
	}
	n--
	if n > 0 && line[n-1] == '\r' {
		n--
	}
	return line[:n]
}
