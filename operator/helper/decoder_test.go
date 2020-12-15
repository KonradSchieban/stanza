package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodings(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		contents []byte
		encoding string
		expected string
	}{
		{
			"Nop",
			[]byte{0xc5},
			"",
			string([]byte{0xc5}),
		},
		{
			"ValidUTF8",
			[]byte("foo"),
			"utf8",
			"foo",
		},
		{
			"UTF8-LessThan-GreaterThan",
			[]byte("\u003cfoo\u003e"),
			"utf8",
			"<foo>",
		},
		{
			"ChineseCharacter",
			[]byte{230, 138, 152},
			"utf8",
			"折",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			decoderCfg := NewDecoderConfig(tc.encoding)
			decoder, err := decoderCfg.Build()
			require.NoError(t, err, "invalid decoder config")

			actual, err := decoder.Decode(tc.contents)
			require.NoError(t, err, "decoding failed")

			require.Equal(t, tc.expected, actual)
		})
	}
}
