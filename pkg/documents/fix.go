package documents

import (
	"os"
	"strings"

	"github.com/goodblaster/errors"
)

// FixContentLocation rewrites the first Content-Location header in content
// to the given location. When an MHT is opened and re-exported using
// ChromeDP, the Content-Location header is changed to the local file path of
// the MHT file, which breaks rendering; this restores the original URL.
func FixContentLocation(content string, location string) string {
	lines := strings.Split(content, "\r\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "Content-Location:") {
			lines[i] = "Content-Location: " + location
			break
		}
	}
	return strings.Join(lines, "\r\n")
}

// ExtractContentLocation returns the value of the first Content-Location
// header in the named MHT file.
func ExtractContentLocation(filename string) (string, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return "", errors.Wrap(err, "failed to read file")
	}

	for line := range strings.SplitSeq(string(b), "\r\n") {
		if strings.HasPrefix(line, "Content-Location:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Content-Location:")), nil
		}
	}

	return "", errors.New("content location not found")
}
