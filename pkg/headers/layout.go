package headers

import (
	"io"
	"net/textproto"
	"slices"
	"strings"

	"github.com/goodblaster/errors"
)

// Layout records how a header block was written in its source: the order of
// the headers, the spelling of their names, and where their values were
// folded. WriteAll uses it to reproduce untouched headers exactly.
type Layout struct {
	fields []layoutField
}

type layoutField struct {
	name  string // name as spelled in the source
	key   string // canonical MIME header key
	value string // unfolded value, as stored in the MIMEHeader
	raw   string // source lines, each terminated by CRLF
}

// WriteAll writes a whole header block (without the terminating blank line).
//
// Headers whose values are unchanged since layout was recorded are written
// exactly as they appeared in the source. A header whose values have changed
// is regenerated in its original position, a removed header is dropped, and
// a new header is placed according to OrderedKeys. With a nil layout every
// header is generated, in Ordered order.
func WriteAll(writer io.Writer, headers textproto.MIMEHeader, layout *Layout) error {
	type entry struct {
		rank int
		text string
	}

	original := make(map[string][]string)
	if layout != nil {
		for _, field := range layout.fields {
			original[field.key] = append(original[field.key], field.value)
		}
	}

	var entries []entry
	regenerated := make(map[string]bool)

	if layout != nil {
		for _, field := range layout.fields {
			current := headers[field.key]
			if slices.Equal(current, original[field.key]) {
				entries = append(entries, entry{rank(field.key), field.raw})
				continue
			}

			// Changed: all current values replace the first occurrence.
			if regenerated[field.key] {
				continue
			}
			regenerated[field.key] = true
			if len(current) > 0 {
				entries = append(entries, entry{rank(field.key), generate(field.name, current)})
			}
		}
	}

	// Headers added since the layout was recorded go ahead of the first
	// well-known header that Blink would write after them; headers Blink
	// does not order go last.
	unknown := len(OrderedKeys())
	_ = Ordered(headers, func(key string, values []string) error {
		if _, ok := original[textproto.CanonicalMIMEHeaderKey(key)]; ok {
			return nil
		}
		r := rank(key)
		at := slices.IndexFunc(entries, func(e entry) bool { return e.rank > r && e.rank < unknown })
		if at < 0 {
			at = len(entries)
		}
		entries = slices.Insert(entries, at, entry{r, generate(key, values)})
		return nil
	})

	for _, e := range entries {
		if _, err := io.WriteString(writer, e.text); err != nil {
			return errors.Wrap(err, "failed to write header")
		}
	}
	return nil
}

// generate renders one header in Blink's style.
func generate(key string, values []string) string {
	var text strings.Builder
	_ = Write(&text, key, values) // a strings.Builder never fails
	return text.String()
}

// rank is the position of key in OrderedKeys; unknown keys sort last.
func rank(key string) int {
	keys := OrderedKeys()
	for i, known := range keys {
		if strings.EqualFold(known, key) {
			return i
		}
	}
	return len(keys)
}
