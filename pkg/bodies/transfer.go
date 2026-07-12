// Package bodies decodes MHT section bodies according to their
// Content-Transfer-Encoding.
package bodies

// TransferEncoding is the value of a section's Content-Transfer-Encoding
// header.
type TransferEncoding string

// Transfer encodings that receive special handling. Anything else is
// treated as an identity encoding and passed through unchanged.
const (
	QuotedPrintable TransferEncoding = "quoted-printable"
	Base64          TransferEncoding = "base64"
)
