// Command mht parses an MHT file and writes it back out, optionally
// extracting its images along the way. Because parsing and writing are
// byte-faithful for Blink-generated files, the default mode doubles as a
// verification that a file survives a round trip.
package main

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/goodblaster/go-mht/pkg/documents"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole program; it returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("mht", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "", "output file (default stdout)")
	imagesDir := flags.String("images", "", "also extract images into this directory")
	flags.Usage = func() {
		fmt.Fprintf(stderr, "Usage: mht [-o out.mht] [-images dir] input.mht\n\n")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}
	input := flags.Arg(0)

	doc, err := documents.ParseFile(input)
	if err != nil {
		fmt.Fprintf(stderr, "parse %s: %v\n", input, err)
		return 1
	}

	if *imagesDir != "" {
		if err := extractImages(doc, *imagesDir, stderr); err != nil {
			fmt.Fprintf(stderr, "extract images: %v\n", err)
			return 1
		}
	}

	if *output == "" {
		if err := doc.Write(stdout); err != nil {
			fmt.Fprintf(stderr, "write: %v\n", err)
			return 1
		}
		return 0
	}

	if err := writeFile(doc, *output); err != nil {
		fmt.Fprintf(stderr, "write %s: %v\n", *output, err)
		return 1
	}
	return 0
}

// writeFile writes doc to the named file, reporting a failed close as well
// as a failed write, since buffered data can be lost at either point.
func writeFile(doc *documents.Document, name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := doc.Write(file); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func extractImages(doc *documents.Document, dir string, stderr io.Writer) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	used := make(map[string]bool)
	for i, section := range doc.ImageSections() {
		data, err := section.ImageBytes()
		if err != nil {
			fmt.Fprintf(stderr, "skipping image %d: %v\n", i, err)
			continue
		}

		name := imageFilename(section.Headers.Get("Content-Location"), section.Headers.Get("Content-Type"), i)
		name = uniqueName(name, used)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}

	return nil
}

// imageFilename derives a file name from the last path segment of the
// section's Content-Location, ignoring any query string or fragment. It
// falls back to an index plus the Content-Type subtype when the location
// has no usable name.
func imageFilename(location, contentType string, index int) string {
	locationPath := location
	if parsed, err := url.Parse(location); err == nil {
		locationPath = parsed.Path
		if locationPath == "" {
			// Opaque locations such as "cid:image001@example".
			locationPath = parsed.Opaque
		}
	} else if cut := strings.IndexAny(locationPath, "?#"); cut >= 0 {
		locationPath = locationPath[:cut]
	}

	// Treat backslashes as separators too, so a name can never carry a
	// directory component on any platform.
	locationPath = strings.ReplaceAll(locationPath, `\`, "/")
	name := sanitize(path.Base(strings.TrimRight(locationPath, "/")))

	if name == "" || name == "." || name == ".." {
		ext := "img"
		if mediaType, _, _ := strings.Cut(contentType, ";"); mediaType != "" {
			if _, subtype, found := strings.Cut(strings.TrimSpace(mediaType), "/"); found && sanitize(subtype) != "" {
				ext = sanitize(subtype)
			}
		}
		name = fmt.Sprintf("image-%d.%s", index, ext)
	}
	return name
}

// sanitize replaces characters that are unsafe or invalid in file names.
func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		if r < ' ' || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, name)
}

// uniqueName returns name, or name with a numeric suffix before its
// extension if name was already handed out, and records the result in used.
// Names are compared case-insensitively, as many file systems do.
func uniqueName(name string, used map[string]bool) string {
	candidate := name
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for n := 1; used[strings.ToLower(candidate)]; n++ {
		candidate = fmt.Sprintf("%s-%d%s", stem, n, ext)
	}
	used[strings.ToLower(candidate)] = true
	return candidate
}
