// Package bodies decodes MHT section bodies according to their
// Content-Transfer-Encoding.
package bodies

import "strings"

// TransferEncoding is the value of a section's Content-Transfer-Encoding
// header.
type TransferEncoding string

// Transfer encodings that receive special handling. Anything else is
// treated as an identity encoding and passed through unchanged.
const (
	QuotedPrintable TransferEncoding = "quoted-printable"
	Base64          TransferEncoding = "base64"
)

// ParseTransferEncoding normalizes a Content-Transfer-Encoding header value.
// The value is case-insensitive, so "Base64" and "BASE64" both yield Base64.
func ParseTransferEncoding(value string) TransferEncoding {
	return TransferEncoding(strings.ToLower(strings.TrimSpace(value)))
}
