package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleFile = "../../pkg/documents/testdata/sample.mht"

func TestImageFilename(t *testing.T) {
	tests := []struct {
		name        string
		location    string
		contentType string
		want        string
	}{
		{"plain", "https://example.com/img/logo.png", "image/png", "logo.png"},
		{"query string dropped", "https://example.com/logo.png?v=2&size=1/2", "image/png", "logo.png"},
		{"fragment dropped", "https://example.com/logo.png#top", "image/png", "logo.png"},
		{"percent-decoded", "https://example.com/my%20logo.png", "image/png", "my logo.png"},
		{"encoded slash cannot add a directory", "https://example.com/a%2Fb.png", "image/png", "b.png"},
		{"backslash cannot add a directory", `https://example.com/..%5C..%5Cevil.png`, "image/png", "evil.png"},
		{"dot-dot falls back", "https://example.com/..", "image/png", "image-3.png"},
		{"no path falls back", "https://example.com/", "image/jpeg", "image-3.jpeg"},
		{"empty location falls back", "", "image/gif", "image-3.gif"},
		{"content type parameters ignored", "", "image/svg+xml; charset=utf-8", "image-3.svg+xml"},
		{"no content type", "", "", "image-3.img"},
		{"cid location", "cid:image001.png@01D0", "image/png", "image001.png@01D0"},
		{"invalid characters replaced", "https://example.com/a:b*c.png", "image/png", "a_b_c.png"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageFilename(tt.location, tt.contentType, 3); got != tt.want {
				t.Errorf("imageFilename(%q, %q) = %q, want %q", tt.location, tt.contentType, got, tt.want)
			}
		})
	}
}

func TestUniqueName(t *testing.T) {
	used := make(map[string]bool)
	var got []string
	for _, name := range []string{"logo.png", "logo.png", "LOGO.png", "logo-1.png", "noext", "noext"} {
		got = append(got, uniqueName(name, used))
	}

	want := []string{"logo.png", "logo-1.png", "LOGO-2.png", "logo-1-1.png", "noext", "noext-1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("names = %v, want %v", got, want)
	}
}

func TestRunRoundTrip(t *testing.T) {
	original, err := os.ReadFile(sampleFile)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	t.Run("stdout", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := run([]string{sampleFile}, &stdout, &stderr); code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
		}
		if !bytes.Equal(stdout.Bytes(), original) {
			t.Error("stdout is not byte-identical to the input")
		}
	})

	t.Run("output file and images", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "out.mht")
		images := filepath.Join(dir, "imgs")

		var stdout, stderr bytes.Buffer
		if code := run([]string{"-o", out, "-images", images, sampleFile}, &stdout, &stderr); code != 0 {
			t.Fatalf("exit code = %d, stderr: %s", code, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Errorf("unexpected stdout output: %q", stdout.String())
		}

		written, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		if !bytes.Equal(written, original) {
			t.Error("output file is not byte-identical to the input")
		}

		png, err := os.ReadFile(filepath.Join(images, "pixel.png"))
		if err != nil {
			t.Fatalf("read extracted image: %v", err)
		}
		if !bytes.HasPrefix(png, []byte("\x89PNG\r\n")) {
			t.Errorf("extracted file is not a PNG: %q", png[:min(8, len(png))])
		}
	})
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want int
	}{
		{"no arguments", nil, 2},
		{"unknown flag", []string{"-nope", sampleFile}, 2},
		{"missing input", []string{filepath.Join(t.TempDir(), "absent.mht")}, 1},
		{"unwritable output", []string{"-o", filepath.Join(t.TempDir(), "no-such-dir", "out.mht"), sampleFile}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(tt.args, &stdout, &stderr); code != tt.want {
				t.Errorf("exit code = %d, want %d (stderr: %s)", code, tt.want, stderr.String())
			}
			if stderr.Len() == 0 {
				t.Error("expected a message on stderr")
			}
		})
	}
}
