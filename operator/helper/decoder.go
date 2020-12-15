package helper

import (
	"fmt"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// NewDecoderConfig creates a new decoder config with default values
func NewDecoderConfig(e string) DecoderConfig {
	return DecoderConfig{
		Encoding: e,
	}
}

// DecoderConfig is the configuration of a decoder
type DecoderConfig struct {
	Encoding string `json:"encoding" yaml:"encoding"`
}

// IsZero returns true if the encoding is 'nop', false otherwise
func (c DecoderConfig) IsZero() bool {
	return c.Encoding == "nop"
}

// Build will build a decoder from the supplied configuration
func (c DecoderConfig) Build() (*Decoder, error) {
	decoder := Decoder{
		Encoding:     encoding.Nop,
		decoder:      encoding.Nop.NewDecoder(),
		decodeBuffer: make([]byte, 1<<12),
	}

	enc, err := lookupEncoding(c.Encoding)
	if err != nil {
		return &decoder, err
	}

	decoder.Encoding = enc

	return &decoder, nil
}

func lookupEncoding(enc string) (encoding.Encoding, error) {
	if encoding, ok := encodingOverrides[strings.ToLower(enc)]; ok {
		return encoding, nil
	}
	encoding, err := ianaindex.IANA.Encoding(enc)
	if err != nil {
		return nil, fmt.Errorf("unsupported encoding '%s'", enc)
	}
	if encoding == nil {
		return nil, fmt.Errorf("no charmap defined for encoding '%s'", enc)
	}
	return encoding, nil
}

var encodingOverrides = map[string]encoding.Encoding{
	"utf-16":   unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf16":    unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
	"utf8":     unicode.UTF8,
	"ascii":    unicode.UTF8,
	"us-ascii": unicode.UTF8,
	"nop":      encoding.Nop,
	"":         encoding.Nop,
}

// Decoder is a helper that adds labels to an entry
type Decoder struct {
	Encoding     encoding.Encoding
	decoder      *encoding.Decoder
	decodeBuffer []byte
}

// Copy copies the decoder
func (d *Decoder) Copy() *Decoder {
	return &Decoder{
		Encoding:     d.Encoding,
		decoder:      d.Encoding.NewDecoder(),
		decodeBuffer: make([]byte, 1<<12),
	}
}

// Decode converts the bytes in msgBuf to utf-8 from the configured encoding
func (d *Decoder) Decode(msgBuf []byte) (string, error) {
	for {
		d.decoder.Reset()
		nDst, _, err := d.decoder.Transform(d.decodeBuffer, msgBuf, true)
		if err != nil && err == transform.ErrShortDst {
			d.decodeBuffer = make([]byte, len(d.decodeBuffer)*2)
			continue
		} else if err != nil {
			return "", fmt.Errorf("transform encoding: %s", err)
		}
		return string(d.decodeBuffer[:nDst]), nil
	}
}
