package lines

import (
	"bufio"
	"strings"
	"testing"
)

func TestSplitKeepsTerminators(t *testing.T) {
	input := "one\r\ntwo\n\r\nlast"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(Split)

	var raw, trimmed []string
	for scanner.Scan() {
		raw = append(raw, scanner.Text())
		trimmed = append(trimmed, Trim(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan error: %v", err)
	}

	if got := strings.Join(raw, ""); got != input {
		t.Errorf("rejoined lines = %q, want %q", got, input)
	}
	want := []string{"one", "two", "", "last"}
	if strings.Join(trimmed, "|") != strings.Join(want, "|") {
		t.Errorf("trimmed lines = %q, want %q", trimmed, want)
	}
}

func TestTrim(t *testing.T) {
	tests := map[string]string{
		"":         "",
		"a":        "a",
		"a\n":      "a",
		"a\r\n":    "a",
		"a\r":      "a\r",
		"a\r\r\n":  "a\r",
		"\r\n":     "",
		"a\n\r\n":  "a\n",
		"a \t\r\n": "a \t",
	}
	for input, want := range tests {
		if got := Trim(input); got != want {
			t.Errorf("Trim(%q) = %q, want %q", input, got, want)
		}
	}
}
