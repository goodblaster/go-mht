package documents

import (
	"image"
	"strings"

	"github.com/goodblaster/errors"
	"github.com/goodblaster/go-mht/pkg/sections"
)

// ImageSections returns every section whose Content-Type is an image type.
func (doc *Document) ImageSections() []sections.Section {
	var sectionList []sections.Section
	for _, section := range doc.Sections {
		if strings.HasPrefix(section.Headers.Get("Content-Type"), "image/") {
			sectionList = append(sectionList, section)
		}
	}
	return sectionList
}

// ImageSection returns the section at the given Content-Location, or nil if
// there is none.
func (doc *Document) ImageSection(location string) *sections.Section {
	return doc.LocIndex[location]
}

// Image decodes the image at the given Content-Location, returning the image
// and its format name (e.g. "png").
func (doc *Document) Image(location string) (image.Image, string, error) {
	section := doc.LocIndex[location]
	if section == nil {
		return nil, "", errors.New("image section not found")
	}
	return section.Image()
}

// ImageBytes returns the raw decoded bytes of the image at the given
// Content-Location.
func (doc *Document) ImageBytes(location string) ([]byte, error) {
	section := doc.LocIndex[location]
	if section == nil {
		return nil, errors.New("image section not found")
	}
	return section.ImageBytes()
}
