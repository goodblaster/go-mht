# go-mht

An MHT (MHTML) parser and writer for Go, built for **byte-faithful round trips**
of Chrome/Blink-generated snapshots.

## Why

Chrome's "Save as MHTML" (and ChromeDP's `page.CaptureSnapshot`) produce MHTML
files with specific conventions: a fixed header order, tab-folded header
values, quoted-printable HTML bodies, and 76-column base64 for binary parts.

Go's standard library (`net/mail` + `mime/multipart` + `mime/quotedprintable`)
can *parse* these files, but it cannot write them back faithfully: headers get
reordered and canonicalized, and quoted-printable content re-encodes
differently. If your workflow is *parse → modify → write* — for example,
swapping an image inside a snapshot — the output drifts from the original and
can render differently.

go-mht parses with a purpose-built scanner that preserves Blink's conventions.
Parsing a Chrome-generated MHT and writing it back out produces a
**byte-identical** file (this is enforced by a golden test).

It also handles a ChromeDP quirk: when an MHT is opened from disk and
re-exported, Blink rewrites the root `Content-Location` to the local file
path, which breaks rendering. `documents.FixContentLocation` restores the
original URL.

## Install

```sh
go get github.com/goodblaster/go-mht
```

## Library usage

### Parse and write

```go
doc, err := documents.ParseFile("snapshot.mht")
if err != nil {
    log.Fatal(err)
}

out, _ := os.Create("copy.mht")
defer out.Close()
if err := doc.Write(out); err != nil {
    log.Fatal(err)
}
// copy.mht is byte-identical to snapshot.mht
```

### Work with images

```go
// All image sections (pointers into the document, so edits stick).
for _, s := range doc.ImageSections() {
    fmt.Println(s.Headers.Get("Content-Location"))
}

// Decode one image by its Content-Location.
img, format, err := doc.Image("https://example.com/logo.png")

// Or grab the raw bytes (works for SVG too, which image.Decode can't read).
data, err := doc.ImageBytes("https://example.com/logo.png")

// Replace an image in place, then write the document back out.
section := doc.ImageSection("https://example.com/logo.png")
err = section.ReplaceImage("image/png", "base64", newPNGBytes)
```

Lookups by `Content-Location` go through `doc.LocIndex`. If you add, remove,
or reorder `doc.Sections` yourself, call `doc.Reindex()` afterwards.

### What is preserved

Headers you don't touch are written back exactly as they were read: same
order, same spelling, same folding. A header you change is regenerated in
place, and a header you add is placed where Blink would put it. Bodies are
re-encoded with a port of Blink's own quoted-printable rules, and binary
parts are carried through byte for byte.

### Fix a ChromeDP re-export

```go
location, err := documents.ExtractContentLocation("original.mht")
// ... re-export via ChromeDP, then:
fixed := documents.FixContentLocation(reExportedContent, location)
```

## CLI

```sh
go install github.com/goodblaster/go-mht/cmd/mht@latest

mht snapshot.mht                     # parse and write to stdout
mht -o out.mht snapshot.mht          # parse and write to a file
mht -images ./imgs -o out.mht in.mht # also extract all images
```

Extracted images are named after the last path segment of their
`Content-Location` (query strings dropped); name clashes get a numeric
suffix rather than overwriting each other.

Since output is byte-faithful, `mht -o out.mht in.mht && cmp in.mht out.mht`
doubles as a check that a file survives a round trip.

## Scope and limitations

- Tuned for Blink-generated MHTML (`From: <Saved by Blink>`). Files from
  other producers (e.g. Word, IE) may parse, but byte-identical output is
  only expected for Blink conventions.
- Supported transfer encodings: `quoted-printable` and `base64`; other
  encodings (`7bit`, `8bit`, `binary`) are passed through unchanged.
- A part with no `Content-Transfer-Encoding` header is read as
  quoted-printable if it decodes cleanly and kept verbatim otherwise. Either
  way it is written back as quoted-printable with the header added, so such
  a part is not byte-identical after a round trip.
- Line endings are expected to be CRLF, as Blink produces. LF-only files
  parse, but headers and text parts are written back with CRLF.
- Base64 bodies are re-wrapped at 76 columns, and the preamble before the
  first boundary is written as a single blank line, both as Blink does.

## License

MIT — see [LICENSE](LICENSE).
