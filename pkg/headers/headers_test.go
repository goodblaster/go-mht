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

func TestWriteOnlyFoldsContentType(t *testing.T) {
	var buf strings.Builder
	if err := Write(&buf, "Subject", []string{"Foo; bar"}); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if want := "Subject: Foo; bar\r\n"; buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

// scanBlock scans a header block and returns the headers and their layout.
func scanBlock(t *testing.T, block string) (textproto.MIMEHeader, *Layout) {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(block + "\r\n"))
	headers := make(textproto.MIMEHeader)
	layout, err := ScanLayout(scanner, &headers)
	if err != nil {
		t.Fatalf("ScanLayout error: %v", err)
	}
	return headers, layout
}

func writeAll(t *testing.T, headers textproto.MIMEHeader, layout *Layout) string {
	t.Helper()

	var buf strings.Builder
	if err := WriteAll(&buf, headers, layout); err != nil {
		t.Fatalf("WriteAll error: %v", err)
	}
	return buf.String()
}

func TestWriteAllReproducesSource(t *testing.T) {
	blocks := map[string]string{
		"blink": "From: <Saved by Blink>\r\n" +
			"Snapshot-Content-Location: https://example.com/\r\n" +
			"Subject: Example\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: multipart/related;\r\n\ttype=\"text/html\";\r\n\tboundary=\"b\"\r\n",
		"semicolon in subject":       "Subject: Foo; bar\r\n",
		"subject folded by blink":    "Subject: =?utf-8?Q?aaa?=\r\n =?utf-8?Q?bbb?=\r\n",
		"content-type on one line":   "Content-Type: text/html; charset=utf-8\r\n",
		"unusual key case":           "X-foo-BAR: 1\r\n",
		"unusual order":              "Content-Location: https://example.com/\r\nContent-Type: text/html\r\n",
		"repeated header":            "X-A: 1\r\nX-B: 2\r\nX-A: 3\r\n",
		"no space after colon":       "X-A:1\r\n",
		"extra whitespace preserved": "X-A:   1  \r\n",
	}

	for name, block := range blocks {
		t.Run(name, func(t *testing.T) {
			headers, layout := scanBlock(t, block)
			if got := writeAll(t, headers, layout); got != block {
				t.Errorf("output = %q, want %q", got, block)
			}
		})
	}
}

func TestWriteAllAfterChanges(t *testing.T) {
	const block = "Content-Type: image/png\r\n" +
		"X-foo-BAR: keep; me\r\n" +
		"Content-Location: https://example.com/a.png\r\n"

	t.Run("changed header is regenerated in place", func(t *testing.T) {
		headers, layout := scanBlock(t, block)
		headers.Set("Content-Type", "image/jpeg")

		want := "Content-Type: image/jpeg\r\n" +
			"X-foo-BAR: keep; me\r\n" +
			"Content-Location: https://example.com/a.png\r\n"
		if got := writeAll(t, headers, layout); got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("new header takes its Blink position", func(t *testing.T) {
		headers, layout := scanBlock(t, block)
		headers.Set("Content-Transfer-Encoding", "base64")
		headers.Set("X-New", "1")

		want := "Content-Type: image/png\r\n" +
			"X-foo-BAR: keep; me\r\n" +
			"Content-Transfer-Encoding: base64\r\n" +
			"Content-Location: https://example.com/a.png\r\n" +
			"X-New: 1\r\n"
		if got := writeAll(t, headers, layout); got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("removed header is dropped", func(t *testing.T) {
		headers, layout := scanBlock(t, block)
		headers.Del("X-Foo-Bar")

		want := "Content-Type: image/png\r\n" +
			"Content-Location: https://example.com/a.png\r\n"
		if got := writeAll(t, headers, layout); got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})

	t.Run("nil layout uses Blink order", func(t *testing.T) {
		headers, _ := scanBlock(t, block)

		want := "Content-Type: image/png\r\n" +
			"Content-Location: https://example.com/a.png\r\n" +
			"X-Foo-Bar: keep; me\r\n"
		if got := writeAll(t, headers, nil); got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
	})
}
