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
// All image sections.
for _, s := range doc.ImageSections() {
    fmt.Println(s.Headers.Get("Content-Location"))
}

// Decode one image by its Content-Location.
img, format, err := doc.Image("https://example.com/logo.png")

// Or grab the raw bytes.
data, err := doc.ImageBytes("https://example.com/logo.png")

// Replace an image in place, then write the document back out.
section := doc.ImageSection("https://example.com/logo.png")
err = section.ReplaceImage("image/png", "base64", newPNGBytes)
```

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

Since output is byte-faithful, `mht -o out.mht in.mht && cmp in.mht out.mht`
doubles as a check that a file survives a round trip.

## Scope and limitations

- Tuned for Blink-generated MHTML (`From: <Saved by Blink>`). Files from
  other producers (e.g. Word, IE) may parse, but byte-identical output is
  only expected for Blink conventions.
- Supported transfer encodings: `quoted-printable` (the default when the
  header is absent) and `base64`; other encodings are passed through
  unchanged.
- Line endings are expected to be CRLF, as Blink produces.

## License

MIT — see [LICENSE](LICENSE).
