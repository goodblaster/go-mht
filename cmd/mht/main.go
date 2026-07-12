// Command mht parses an MHT file and writes it back out, optionally
// extracting its images along the way. Because parsing and writing are
// byte-faithful for Blink-generated files, the default mode doubles as a
// verification that a file survives a round trip.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/goodblaster/go-mht/pkg/documents"
)

func main() {
	output := flag.String("o", "", "output file (default stdout)")
	imagesDir := flag.String("images", "", "also extract images into this directory")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mht [-o out.mht] [-images dir] input.mht\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	doc, err := documents.ParseFile(flag.Arg(0))
	if err != nil {
		fatal("parse %s: %v", flag.Arg(0), err)
	}

	if *imagesDir != "" {
		if err := extractImages(doc, *imagesDir); err != nil {
			fatal("extract images: %v", err)
		}
	}

	var out io.Writer = os.Stdout
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			fatal("create %s: %v", *output, err)
		}
		defer file.Close()
		out = file
	}

	if err := doc.Write(out); err != nil {
		fatal("write: %v", err)
	}
}

func extractImages(doc *documents.Document, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for i, section := range doc.ImageSections() {
		data, err := section.ImageBytes()
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping image %d: %v\n", i, err)
			continue
		}

		name := imageFilename(section.Headers.Get("Content-Location"), section.Headers.Get("Content-Type"), i)
		if err := os.WriteFile(path.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}

	return nil
}

// imageFilename derives a file name from the section's Content-Location,
// falling back to an index plus the Content-Type subtype.
func imageFilename(location, contentType string, index int) string {
	name := path.Base(strings.TrimSuffix(location, "/"))
	if name == "" || name == "." || name == "/" {
		ext := "img"
		if _, subtype, found := strings.Cut(contentType, "/"); found {
			ext = subtype
		}
		name = fmt.Sprintf("image-%d.%s", index, ext)
	}
	return name
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
