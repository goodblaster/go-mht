package documents

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goodblaster/go-mht/pkg/sections"
)

const sampleFile = "testdata/sample.mht"

// TestRoundTrip is the core invariant of this library: parsing a
// Blink-style MHT file and writing it back out is byte-identical.
func TestRoundTrip(t *testing.T) {
	original, err := os.ReadFile(sampleFile)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	doc, err := ParseFile(sampleFile)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	var out bytes.Buffer
	if err := doc.Write(&out); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	if !bytes.Equal(original, out.Bytes()) {
		t.Errorf("round trip is not byte-identical (got %d bytes, want %d)", out.Len(), len(original))
	}
}

func TestParse(t *testing.T) {
	doc, err := ParseFile(sampleFile)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	if doc.Boundary != "----MultipartBoundary--abc123XYZ----" {
		t.Errorf("Boundary = %q", doc.Boundary)
	}
	if got := doc.Headers.Get("Snapshot-Content-Location"); got != "https://example.com/" {
		t.Errorf("Snapshot-Content-Location = %q", got)
	}
	if len(doc.Sections) != 3 {
		t.Fatalf("len(Sections) = %d, want 3", len(doc.Sections))
	}

	html := doc.Sections[0]
	if !strings.Contains(html.Body, `<img src="https://example.com/pixel.png">`) {
		t.Errorf("html body not decoded as expected: %q", html.Body)
	}
}

func TestImages(t *testing.T) {
	doc, err := ParseFile(sampleFile)
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}

	if got := len(doc.ImageSections()); got != 1 {
		t.Fatalf("len(ImageSections) = %d, want 1", got)
	}

	img, format, err := doc.Image("https://example.com/pixel.png")
	if err != nil {
		t.Fatalf("Image error: %v", err)
	}
	if format != "png" {
		t.Errorf("format = %q, want png", format)
	}
	if b := img.Bounds(); b.Dx() != 1 || b.Dy() != 1 {
		t.Errorf("bounds = %v, want 1x1", b)
	}

	if _, _, err := doc.Image("https://example.com/missing.png"); err == nil {
		t.Error("Image for unknown location expected error")
	}
}

func TestFixContentLocation(t *testing.T) {
	content := strings.Join([]string{
		"Content-Type: text/html",
		"Content-Location: file:///C:/temp/export.mht",
		"",
		"body",
	}, "\r\n")

	fixed := FixContentLocation(content, "https://example.com/")
	if !strings.Contains(fixed, "Content-Location: https://example.com/\r\n") {
		t.Errorf("FixContentLocation did not rewrite the header: %q", fixed)
	}
}

func TestExtractContentLocation(t *testing.T) {
	got, err := ExtractContentLocation(sampleFile)
	if err != nil {
		t.Fatalf("ExtractContentLocation error: %v", err)
	}
	if got != "https://example.com/" {
		t.Errorf("ExtractContentLocation = %q, want %q", got, "https://example.com/")
	}

	missing := filepath.Join(t.TempDir(), "none.mht")
	if err := os.WriteFile(missing, []byte("no headers here\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractContentLocation(missing); err == nil {
		t.Error("expected error when no Content-Location present")
	}
}

// roundTrip parses input and writes it back out.
func roundTrip(t *testing.T, input string) string {
	t.Helper()

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var out bytes.Buffer
	if err := doc.Write(&out); err != nil {
		t.Fatalf("Write error: %v", err)
	}
	return out.String()
}

// TestRoundTripBlinkPartEndings covers how Blink ends a quoted-printable
// part, which differs from mime/quotedprintable: a final newline is escaped
// as =0A, and the final line may be a full 76 columns.
func TestRoundTripBlinkPartEndings(t *testing.T) {
	const header = "From: <Saved by Blink>\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/related;\r\n" +
		"\ttype=\"text/html\";\r\n" +
		"\tboundary=\"----B----\"\r\n" +
		"\r\n\r\n"
	const partHeader = "------B----\r\n" +
		"Content-Type: text/css\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"Content-Location: https://example.com/a.css\r\n" +
		"\r\n"
	const closing = "------B------\r\n"

	tests := []struct {
		name string
		body string
	}{
		{"no trailing newline", "a { color: red; }\r\n"},
		{"final newline escaped", "a { color: red; }=0A\r\n"},
		{"blank line then escaped newline", "a { color: red; }\r\n=0A\r\n"},
		{"final line of 76 columns", "b {}\r\n" + strings.Repeat("x", 76) + "\r\n"},
		{"soft-wrapped final line", strings.Repeat("x", 75) + "=\r\n" + strings.Repeat("y", 76) + "\r\n"},
		{"empty body", "\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := header + partHeader + tt.body + partHeader + tt.body + closing
			if got := roundTrip(t, input); got != input {
				t.Errorf("round trip is not byte-identical:\n got %q\nwant %q", got, input)
			}
		})
	}
}

// TestReplaceImageIsWritten checks that a replacement made through the
// document's lookups reaches the written output, body included.
func TestReplaceImageIsWritten(t *testing.T) {
	const location = "https://example.com/pixel.png"
	lookups := map[string]func(doc *Document) *sections.Section{
		"ImageSection":  func(doc *Document) *sections.Section { return doc.ImageSection(location) },
		"ImageSections": func(doc *Document) *sections.Section { return doc.ImageSections()[0] },
	}

	for name, lookup := range lookups {
		t.Run(name, func(t *testing.T) {
			doc, err := ParseFile(sampleFile)
			if err != nil {
				t.Fatalf("ParseFile error: %v", err)
			}

			section := lookup(doc)
			if section == nil {
				t.Fatal("section not found")
			}
			if section != &doc.Sections[2] {
				t.Fatal("lookup did not return the document's own section")
			}
			if err := section.ReplaceImage("image/jpeg", "base64", []byte("new image bytes")); err != nil {
				t.Fatalf("ReplaceImage error: %v", err)
			}

			var out bytes.Buffer
			if err := doc.Write(&out); err != nil {
				t.Fatalf("Write error: %v", err)
			}

			reparsed, err := Parse(&out)
			if err != nil {
				t.Fatalf("re-Parse error: %v", err)
			}
			if got := reparsed.ImageSection(location).Headers.Get("Content-Type"); got != "image/jpeg" {
				t.Errorf("Content-Type = %q, want image/jpeg", got)
			}
			got, err := reparsed.ImageBytes(location)
			if err != nil {
				t.Fatalf("ImageBytes error: %v", err)
			}
			if string(got) != "new image bytes" {
				t.Errorf("image bytes = %q, want the replacement", got)
			}
		})
	}
}

const (
	testHeader = "From: <Saved by Blink>\r\n" +
		"Subject: Test\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: multipart/related;\r\n" +
		"\ttype=\"text/html\";\r\n" +
		"\tboundary=\"----B----\"\r\n" +
		"\r\n\r\n"
	testBoundary = "------B----\r\n"
	testClosing  = "------B------\r\n"
)

func TestRoundTripHeaders(t *testing.T) {
	part := testBoundary +
		"Content-Type: text/html\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"Content-Location: https://example.com/a;b=1\r\n" +
		"\r\n" +
		"<p>hi</p>\r\n"

	subjects := map[string]string{
		"semicolon in title":      "Subject: Foo; bar\r\n",
		"title folded by blink":   "Subject: =?utf-8?Q?=E3=81=82=E3=81=84=E3=81=86?=\r\n =?utf-8?Q?=E3=81=88=E3=81=8A?=\r\n",
		"unknown header key case": "Subject: Test\r\nX-foo-BAR: 1; 2\r\n",
	}

	for name, subject := range subjects {
		t.Run(name, func(t *testing.T) {
			input := strings.Replace(testHeader, "Subject: Test\r\n", subject, 1) + part + testClosing
			if got := roundTrip(t, input); got != input {
				t.Errorf("round trip is not byte-identical:\n got %q\nwant %q", got, input)
			}
		})
	}
}

func TestRoundTripBinaryPart(t *testing.T) {
	// Bytes that a line-oriented reader would mangle: lone LF, lone CR,
	// CRLF, NUL, and high bytes.
	const data = "\x89PNG\r\n\x1a\n\x00\x01lone\nLF and lone\rCR\xff\xfe\r\n\r\ntail"

	input := testHeader +
		testBoundary +
		"Content-Type: image/png\r\n" +
		"Content-Transfer-Encoding: binary\r\n" +
		"Content-Location: https://example.com/a.png\r\n" +
		"\r\n" +
		data + "\r\n" +
		testClosing

	doc, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got := doc.Sections[0].Body; got != data {
		t.Errorf("Body = %q, want %q", got, data)
	}
	if got := roundTrip(t, input); got != input {
		t.Errorf("round trip is not byte-identical:\n got %q\nwant %q", got, input)
	}
}

func TestParseIgnoresEmptyTrailingSection(t *testing.T) {
	part := testBoundary +
		"Content-Type: text/html\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"<p>hi</p>\r\n"

	inputs := map[string]string{
		"boundary then end of input": testHeader + part + testBoundary,
		"no closing boundary":        testHeader + part,
		"blank lines after boundary": testHeader + part + testBoundary + "\r\n\r\n",
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			doc, err := Parse(strings.NewReader(input))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if len(doc.Sections) != 1 {
				t.Fatalf("len(Sections) = %d, want 1", len(doc.Sections))
			}
			if got := doc.Sections[0].Body; got != "<p>hi</p>" {
				t.Errorf("Body = %q", got)
			}
		})
	}
}

// TestParseUndeclaredEncoding covers parts with no
// Content-Transfer-Encoding header: quoted-printable is assumed, but content
// that is not valid quoted-printable must not fail the parse.
func TestParseUndeclaredEncoding(t *testing.T) {
	partHeader := testBoundary +
		"Content-Type: text/css\r\n" +
		"Content-Location: https://example.com/a.css\r\n" +
		"\r\n"

	tests := []struct {
		name     string
		encoded  string
		wantBody string
	}{
		{"quoted-printable content", "a =3D b\r\n", "a = b"},
		{"plain content with a bare equals sign", "a = b\r\nc =3D d\r\n", "a = b\r\nc =3D d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := Parse(strings.NewReader(testHeader + partHeader + tt.encoded + testClosing))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if got := doc.Sections[0].Body; got != tt.wantBody {
				t.Errorf("Body = %q, want %q", got, tt.wantBody)
			}

			// Writing declares the encoding; the content must survive.
			var out bytes.Buffer
			if err := doc.Write(&out); err != nil {
				t.Fatalf("Write error: %v", err)
			}
			if !strings.Contains(out.String(), "Content-Type: text/css\r\nContent-Transfer-Encoding: quoted-printable\r\nContent-Location:") {
				t.Errorf("encoding not declared in its Blink position:\n%s", out.String())
			}
			if _, declared := doc.Sections[0].Headers["Content-Transfer-Encoding"]; declared {
				t.Error("Write modified the section's headers")
			}

			reparsed, err := Parse(&out)
			if err != nil {
				t.Fatalf("re-Parse error: %v", err)
			}
			if got := reparsed.Sections[0].Body; got != tt.wantBody {
				t.Errorf("Body after rewrite = %q, want %q", got, tt.wantBody)
			}
		})
	}
}

// TestReplaceImageEncodings checks that a replaced image survives a write
// and re-parse whichever transfer encoding it is stored with.
func TestReplaceImageEncodings(t *testing.T) {
	const location = "https://example.com/pixel.png"
	const data = "\x00\x01\n\r\xffraw image bytes"

	for _, encoding := range []string{"base64", "Base64", "binary"} {
		t.Run(encoding, func(t *testing.T) {
			doc, err := ParseFile(sampleFile)
			if err != nil {
				t.Fatalf("ParseFile error: %v", err)
			}
			if err := doc.ImageSection(location).ReplaceImage("image/jpeg", encoding, []byte(data)); err != nil {
				t.Fatalf("ReplaceImage error: %v", err)
			}

			var out bytes.Buffer
			if err := doc.Write(&out); err != nil {
				t.Fatalf("Write error: %v", err)
			}

			reparsed, err := Parse(&out)
			if err != nil {
				t.Fatalf("re-Parse error: %v", err)
			}
			if len(reparsed.Sections) != 3 {
				t.Fatalf("len(Sections) = %d, want 3", len(reparsed.Sections))
			}

			section := reparsed.ImageSection(location)
			if section == nil {
				t.Fatal("replaced section not found after re-parse")
			}
			got := []byte(section.Body)
			if section.IsBase64() {
				if got, err = section.ImageBytes(); err != nil {
					t.Fatalf("ImageBytes error: %v", err)
				}
			}
			if string(got) != data {
				t.Errorf("image bytes = %q, want %q", got, data)
			}
		})
	}
}
