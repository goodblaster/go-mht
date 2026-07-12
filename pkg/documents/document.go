// Package documents parses and writes MHT (MHTML) files, preserving the
// header order, folding, and encoding conventions of Blink-generated
// snapshots so that a parse/write round trip is byte-identical.
package documents

import (
	"net/textproto"

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
}

// New returns an empty Document with initialized headers and indexes.
func New() *Document {
	return &Document{
		Headers:  make(textproto.MIMEHeader),
		Sections: make([]sections.Section, 0),
		LocIndex: make(map[string]*sections.Section),
	}
}
