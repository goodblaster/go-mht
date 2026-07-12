package documents

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if len(doc.Sections) != 2 {
		t.Fatalf("len(Sections) = %d, want 2", len(doc.Sections))
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
