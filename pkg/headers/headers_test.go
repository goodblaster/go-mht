package headers

import (
	"bufio"
	"net/textproto"
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	input := strings.Join([]string{
		"",
		"From: <Saved by Blink>",
		"Content-Type: multipart/related;",
		"\ttype=\"text/html\";",
		"\tboundary=\"XYZ\"",
		"X-Dup: one",
		"X-Dup: two",
		"",
		"body starts here",
	}, "\r\n")

	scanner := bufio.NewScanner(strings.NewReader(input))
	headers := make(textproto.MIMEHeader)
	if err := Scan(scanner, &headers); err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if got := headers.Get("From"); got != "<Saved by Blink>" {
		t.Errorf("From = %q", got)
	}
	if got := headers.Get("Content-Type"); got != `multipart/related; type="text/html"; boundary="XYZ"` {
		t.Errorf("Content-Type = %q", got)
	}
	if got := headers.Values("X-Dup"); len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Errorf("X-Dup values = %v, want [one two]", got)
	}

	// The blank line terminated the block; the body line is still unread.
	if !scanner.Scan() || scanner.Text() != "body starts here" {
		t.Errorf("scanner should be positioned at the body, got %q", scanner.Text())
	}
}

func TestScanSpaceContinuation(t *testing.T) {
	input := "Subject: part one\r\n part two\r\n\r\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	headers := make(textproto.MIMEHeader)
	if err := Scan(scanner, &headers); err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if got := headers.Get("Subject"); got != "part one part two" {
		t.Errorf("Subject = %q", got)
	}
}

func TestScanMalformed(t *testing.T) {
	scanner := bufio.NewScanner(strings.NewReader("this line has no colon\r\n"))
	headers := make(textproto.MIMEHeader)
	if err := Scan(scanner, &headers); err == nil {
		t.Fatal("Scan expected error for malformed header line")
	}
}

func TestOrdered(t *testing.T) {
	headers := textproto.MIMEHeader{}
	headers.Set("Content-Location", "https://example.com/")
	headers.Set("From", "<Saved by Blink>")
	headers.Set("X-Zebra", "z")
	headers.Set("X-Apple", "a")

	var got []string
	err := Ordered(headers, func(key string, values []string) error {
		got = append(got, key)
		return nil
	})
	if err != nil {
		t.Fatalf("Ordered error: %v", err)
	}

	want := []string{"From", "Content-Location", "X-Apple", "X-Zebra"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("key order = %v, want %v", got, want)
	}
}

func TestWriteFolding(t *testing.T) {
	var buf strings.Builder
	value := `multipart/related; type="text/html"; boundary="a;b"`
	if err := Write(&buf, "Content-Type", []string{value}); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	want := "Content-Type: multipart/related;\r\n\ttype=\"text/html\";\r\n\tboundary=\"a;b\"\r\n"
	if buf.String() != want {
		t.Errorf("folded output = %q, want %q", buf.String(), want)
	}
}

func TestWriteContentLocationNotFolded(t *testing.T) {
	var buf strings.Builder
	value := "https://example.com/page;jsessionid=1"
	if err := Write(&buf, "Content-Location", []string{value}); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	want := "Content-Location: " + value + "\r\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}
