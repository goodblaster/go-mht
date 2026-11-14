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

func TestEncodeQuotedPrintable(t *testing.T) {
	a75 := strings.Repeat("a", 75)
	a76 := strings.Repeat("a", 76)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain", "hello", "hello"},
		{"equals sign", "1 + 1 = 2", "1 + 1 =3D 2"},
		{"non-ascii", "caf\xc3\xa9", "caf=C3=A9"},
		{"interior line endings become CRLF", "a\nb\r\nc\rd", "a\r\nb\r\nc\r\nd"},
		{"trailing CRLF is a line break", "a\r\n", "a\r\n"},
		{"final LF is escaped", "a\n", "a=0A"},
		{"final CR is escaped", "a\r", "a=0D"},
		{"only the final LF is escaped", "a\n\n", "a\r\n=0A"},
		{"space before line ending", "a \nb", "a=20\r\nb"},
		{"tab at end of input", "a\t", "a=09"},
		{"interior whitespace untouched", "a \tb", "a \tb"},
		{"final line may fill 76 columns", a76, a76},
		{"soft break leaves room for the equals sign", a76 + "b", a75 + "=\r\na" + "b"},
		{"75 columns before a line break", a75 + "\nb", a75 + "\r\nb"},
		{"76 columns before a line break wraps", a76 + "\nb", a75 + "=\r\na\r\nb"},
		{"escape is never split", strings.Repeat("a", 73) + "\xc3\xa9", strings.Repeat("a", 73) + "=\r\n=C3=A9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EncodeQuotedPrintable(tt.input); got != tt.want {
				t.Errorf("EncodeQuotedPrintable(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestScanUndeclaredEncoding(t *testing.T) {
	const boundary = "--BOUND"

	t.Run("valid quoted-printable is decoded", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader("a =3D b\r\nsoft=\r\nwrap\r\n--BOUND\r\n"))

		var body strings.Builder
		if err := Scan(scanner, boundary, "", &body); err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if want := "a = b\r\nsoftwrap\r\n"; body.String() != want {
			t.Errorf("body = %q, want %q", body.String(), want)
		}
	})

	t.Run("anything else is kept verbatim", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader("a =3D b\r\nx = y\r\nlast\r\n--BOUND\r\n"))

		var body strings.Builder
		if err := Scan(scanner, boundary, "", &body); err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if want := "a =3D b\r\nx = y\r\nlast\r\n"; body.String() != want {
			t.Errorf("body = %q, want %q", body.String(), want)
		}
	})

	t.Run("declared quoted-printable still rejects bad input", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader("x = y\r\n--BOUND\r\n"))

		var body strings.Builder
		if err := Scan(scanner, boundary, QuotedPrintable, &body); err == nil {
			t.Fatal("Scan expected an error for an invalid escape")
		}
	})

	t.Run("encoding name is case-insensitive", func(t *testing.T) {
		scanner := bufio.NewScanner(strings.NewReader("QUJD\r\nREVG\r\n--BOUND\r\n"))

		var body strings.Builder
		if err := Scan(scanner, boundary, "Base64", &body); err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if body.String() != "QUJDREVG" {
			t.Errorf("body = %q, want QUJDREVG", body.String())
		}
	})
}
