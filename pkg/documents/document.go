// Package documents parses and writes MHT (MHTML) files, preserving the
// header order, folding, and encoding conventions of Blink-generated
// snapshots so that a parse/write round trip is byte-identical.
package documents

import (
	"net/textproto"

	"github.com/goodblaster/go-mht/pkg/headers"
	"github.com/goodblaster/go-mht/pkg/sections"
)

// Document is a parsed MHT file: its top-level headers, multipart boundary,
// and sections. LocIndex maps each section's Content-Location to the section
// for direct lookup.
type Document struct {
	Boundary string
	Headers  textproto.MIMEHeader
	Sections []sections.Section
	LocIndex map[string]*sections.Section

	// layout remembers how the top-level headers were written in the
	// source so that untouched headers are written back byte for byte.
	layout *headers.Layout
}

// Reindex rebuilds LocIndex so that it points at the elements of Sections.
// Parse calls it for you; call it again after adding, removing, or
// reordering sections, since those operations invalidate the pointers.
func (doc *Document) Reindex() {
	doc.LocIndex = make(map[string]*sections.Section, len(doc.Sections))
	for i := range doc.Sections {
		section := &doc.Sections[i]
		doc.LocIndex[section.Headers.Get("Content-Location")] = section
	}
}

// New returns an empty Document with initialized headers and indexes.
func New() *Document {
	return &Document{
		Headers:  make(textproto.MIMEHeader),
		Sections: make([]sections.Section, 0),
		LocIndex: make(map[string]*sections.Section),
	}
}
