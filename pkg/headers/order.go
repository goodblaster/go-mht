package headers

import (
	"net/textproto"
	"sort"
	"strings"
)

// OrderedKeys returns the header order used by Blink-generated MHTML files.
// Writing headers in this order keeps output byte-identical to Chrome's.
func OrderedKeys() []string {
	return []string{
		"From",
		"Snapshot-Content-Location",
		"Subject",
		"Date",
		"MIME-Version",
		"Content-Type",
		"Content-ID",
		"Content-Transfer-Encoding",
		"Content-Location",
	}
}

// Ordered calls f for each header in headers: first the well-known keys in
// OrderedKeys order, then any remaining keys sorted alphabetically so output
// is deterministic.
func Ordered(headers textproto.MIMEHeader, f func(key string, values []string) error) error {
	known := make(map[string]bool)
	for _, key := range OrderedKeys() {
		known[strings.ToLower(key)] = true
		if values := headers.Values(key); len(values) > 0 {
			if err := f(key, values); err != nil {
				return err
			}
		}
	}

	var rest []string
	for key := range headers {
		if !known[strings.ToLower(key)] {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)

	for _, key := range rest {
		if err := f(key, headers[key]); err != nil {
			return err
		}
	}

	return nil
}
