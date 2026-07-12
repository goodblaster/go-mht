package sections

import (
	"bytes"
	"encoding/base64"
	"image"
	"strings"

	// Register decoders for the image formats commonly found in MHT files.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/goodblaster/errors"
)

// isImage reports whether the section holds base64-encoded image data,
// returning an error describing what is wrong otherwise.
func (section *Section) isImage() error {
	if !section.IsBase64() {
		return errors.New("section is not base64 encoded")
	}
	if !strings.HasPrefix(section.Headers.Get("Content-Type"), "image/") {
		return errors.New("section is not an image")
	}
	return nil
}

// Image decodes the section body as an image, returning the image and the
// format name reported by image.Decode (e.g. "png").
func (section *Section) Image() (image.Image, string, error) {
	imageBytes, err := section.ImageBytes()
	if err != nil {
		return nil, "", err
	}
	return image.Decode(bytes.NewReader(imageBytes))
}

// ImageBytes returns the section body decoded from base64 into raw image
// bytes.
func (section *Section) ImageBytes() ([]byte, error) {
	if err := section.isImage(); err != nil {
		return nil, err
	}

	imageBytes, err := base64.StdEncoding.DecodeString(section.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode base64 data")
	}

	return imageBytes, nil
}

// ReplaceImage replaces the section's content with imageBytes, updating the
// Content-Type and Content-Transfer-Encoding headers. If contentEncoding is
// "base64" the bytes are encoded; otherwise they are stored verbatim.
func (section *Section) ReplaceImage(contentType, contentEncoding string, imageBytes []byte) error {
	section.Headers.Set("Content-Type", contentType)
	section.Headers.Set("Content-Transfer-Encoding", contentEncoding)

	if contentEncoding == "base64" {
		section.Body = base64.StdEncoding.EncodeToString(imageBytes)
	} else {
		section.Body = string(imageBytes)
	}

	return nil
}
