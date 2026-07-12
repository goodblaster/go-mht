package bodies

import (
	"bufio"
	"strings"
	"testing"
)

func TestDecodeQuotedPrintableLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain line gets hard break", "hello", "hello\r\n"},
		{"empty line is a blank line", "", "\r\n"},
		{"hex escape uppercase", "1 + 1 =3D 2", "1 + 1 = 2\r\n"},
		{"hex escape lowercase", "a=3db", "a=b\r\n"},
		{"soft break drops line ending", "long line=", "long line"},
		{"escape then soft break", "x=3D=", "x="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeQuotedPrintableLine(tt.input)
			if err != nil {
				t.Fatalf("DecodeQuotedPrintableLine(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("DecodeQuotedPrintableLine(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodeQuotedPrintableLineInvalid(t *testing.T) {
	for _, input := range []string{"bad =zz escape", "dangling =A"} {
		if _, err := DecodeQuotedPrintableLine(input); err == nil {
			t.Errorf("DecodeQuotedPrintableLine(%q) expected error, got nil", input)
		}
	}
}

func TestScan(t *testing.T) {
	const boundary = "--BOUND"

	t.Run("quoted-printable stops at boundary", func(t *testing.T) {
		input := "line one\r\n1 =3D one\r\n--BOUND\r\nafter"
		scanner := bufio.NewScanner(strings.NewReader(input))

		var body strings.Builder
		err := Scan(scanner, boundary, QuotedPrintable, &body)
		if err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if want := "line one\r\n1 = one\r\n"; body.String() != want {
			t.Errorf("body = %q, want %q", body.String(), want)
		}
	})

	t.Run("base64 concatenates without breaks", func(t *testing.T) {
		input := "QUJD\r\nREVG\r\n--BOUND--\r\n"
		scanner := bufio.NewScanner(strings.NewReader(input))

		var body strings.Builder
		err := Scan(scanner, boundary, Base64, &body)
		if err == nil {
			t.Fatal("Scan expected io.EOF at closing boundary, got nil")
		}
		if body.String() != "QUJDREVG" {
			t.Errorf("body = %q, want %q", body.String(), "QUJDREVG")
		}
	})

	t.Run("identity encoding passes through", func(t *testing.T) {
		input := "raw = line\r\n--BOUND\r\n"
		scanner := bufio.NewScanner(strings.NewReader(input))

		var body strings.Builder
		if err := Scan(scanner, boundary, "7bit", &body); err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if want := "raw = line\r\n"; body.String() != want {
			t.Errorf("body = %q, want %q", body.String(), want)
		}
	})
}
