package sections

import (
	"strings"
	"testing"
)

func TestWriteDefaultsToQuotedPrintable(t *testing.T) {
	section := New()
	section.Headers.Set("Content-Type", "text/html")
	section.Body = "1 + 1 = 2\r\n"

	var buf strings.Builder
	if err := section.Write(&buf); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Content-Transfer-Encoding: quoted-printable\r\n") {
		t.Errorf("missing default Content-Transfer-Encoding header:\n%s", out)
	}
	if !strings.Contains(out, "1 + 1 =3D 2\r\n") {
		t.Errorf("body not quoted-printable encoded:\n%s", out)
	}
}

func TestWriteBase64Wrapping(t *testing.T) {
	section := New()
	section.Headers.Set("Content-Type", "image/png")
	section.Headers.Set("Content-Transfer-Encoding", "base64")
	section.Body = strings.Repeat("A", 100)

	var buf strings.Builder
	if err := section.Write(&buf); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	_, body, found := strings.Cut(buf.String(), "\r\n\r\n")
	if !found {
		t.Fatalf("no header/body separator in output:\n%s", buf.String())
	}

	want := strings.Repeat("A", 76) + "\r\n" + strings.Repeat("A", 24) + "\r\n\r\n"
	if body != want {
		t.Errorf("base64 body = %q, want %q", body, want)
	}
}

func TestReplaceImage(t *testing.T) {
	section := New()
	section.Headers.Set("Content-Type", "image/png")
	section.Headers.Set("Content-Transfer-Encoding", "base64")
	section.Headers.Set("Content-Location", "https://example.com/a.png")

	raw := []byte{0x01, 0x02, 0x03}
	if err := section.ReplaceImage("image/jpeg", "base64", raw); err != nil {
		t.Fatalf("ReplaceImage error: %v", err)
	}

	if got := section.Headers.Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("Content-Type = %q", got)
	}
	if section.Body != "AQID" {
		t.Errorf("Body = %q, want AQID", section.Body)
	}

	got, err := section.ImageBytes()
	if err != nil {
		t.Fatalf("ImageBytes error: %v", err)
	}
	if string(got) != string(raw) {
		t.Errorf("ImageBytes = %v, want %v", got, raw)
	}
}

func TestImageBytesQuotedPrintable(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg"></svg>`

	section := New()
	section.Headers.Set("Content-Type", "image/svg+xml")
	section.Headers.Set("Content-Transfer-Encoding", "quoted-printable")
	section.Body = svg

	got, err := section.ImageBytes()
	if err != nil {
		t.Fatalf("ImageBytes error: %v", err)
	}
	if string(got) != svg {
		t.Errorf("ImageBytes = %q, want %q", got, svg)
	}
}

func TestImageBytesRejectsNonImage(t *testing.T) {
	section := New()
	section.Headers.Set("Content-Type", "text/html")
	section.Headers.Set("Content-Transfer-Encoding", "base64")
	section.Body = "AQID"

	if _, err := section.ImageBytes(); err == nil {
		t.Fatal("ImageBytes expected an error for a non-image section")
	}
}

func TestWriteDoesNotModifySection(t *testing.T) {
	section := New()
	section.Headers.Set("Content-Type", "text/html")
	section.Body = "hi"

	var buf strings.Builder
	if err := section.Write(&buf); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if len(section.Headers) != 1 {
		t.Errorf("Write modified the headers: %v", section.Headers)
	}
	if want := "Content-Type: text/html\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\nhi\r\n"; buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}
