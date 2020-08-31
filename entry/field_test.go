package entry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestFieldUnmarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			NewRecordField("test1"),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			NewRecordField("test1", "test2"),
		},
		{
			"RootField",
			[]byte(`"$"`),
			NewRecordField([]string{}...),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := json.Unmarshal(tc.input, &f)
			require.NoError(t, err)

			require.Equal(t, tc.expected, f)
		})
	}
}

func TestFieldMarshalJSON(t *testing.T) {
	cases := []struct {
		name     string
		input    Field
		expected []byte
	}{
		{
			"SimpleField",
			NewRecordField("test1"),
			[]byte(`"test1"`),
		},
		{
			"ComplexField",
			NewRecordField("test1", "test2"),
			[]byte(`"test1.test2"`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := json.Marshal(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, res)
		})
	}
}

func TestFieldUnmarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    []byte
		expected Field
	}{
		{
			"SimpleField",
			[]byte(`"test1"`),
			NewRecordField("test1"),
		},
		{
			"UnquotedField",
			[]byte(`test1`),
			NewRecordField("test1"),
		},
		{
			"RootField",
			[]byte(`"$"`),
			NewRecordField([]string{}...),
		},
		{
			"ComplexField",
			[]byte(`"test1.test2"`),
			NewRecordField("test1", "test2"),
		},
		{
			"ComplexFieldWithRoot",
			[]byte(`"$.test1.test2"`),
			NewRecordField("test1", "test2"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var f Field
			err := yaml.UnmarshalStrict(tc.input, &f)
			require.NoError(t, err)

			require.Equal(t, tc.expected, f)
		})
	}
}

func TestFieldMarshalYAML(t *testing.T) {
	cases := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			"SimpleField",
			NewRecordField("test1"),
			"test1\n",
		},
		{
			"ComplexField",
			NewRecordField("test1", "test2"),
			"test1.test2\n",
		},
		{
			"EmptyField",
			NewRecordField(),
			"$record\n",
		},
		{
			"FieldWithDots",
			NewRecordField("test.1"),
			"$record['test.1']\n",
		},
		{
			"FieldWithDotsThenNone",
			NewRecordField("test.1", "test2"),
			"$record['test.1']['test2']\n",
		},
		{
			"FieldWithNoDotsThenDots",
			NewRecordField("test1", "test.2"),
			"$record['test1']['test.2']\n",
		},
		{
			"LabelField",
			NewLabelField("test1"),
			"$labels.test1\n",
		},
		{
			"LabelFieldWithDots",
			NewLabelField("test.1"),
			"$labels['test.1']\n",
		},
		{
			"ResourceField",
			NewResourceField("test1"),
			"$resource.test1\n",
		},
		{
			"ResourceFieldWithDots",
			NewResourceField("test.1"),
			"$resource['test.1']\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := yaml.Marshal(tc.input)
			require.NoError(t, err)

			require.Equal(t, tc.expected, string(res))
		})
	}
}

func TestSplitField(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		output    []string
		expectErr bool
	}{
		{"Simple", "test", []string{"test"}, false},
		{"Sub", "test.case", []string{"test", "case"}, false},
		{"Root", "$", []string{"$"}, false},
		{"RootWithSub", "$record.field", []string{"$record", "field"}, false},
		{"RootWithTwoSub", "$record.field1.field2", []string{"$record", "field1", "field2"}, false},
		{"BracketSyntaxSingleQuote", "['test']", []string{"test"}, false},
		{"BracketSyntaxDoubleQuote", `["test"]`, []string{"test"}, false},
		{"RootSubBracketSyntax", `$record["test"]`, []string{"$record", "test"}, false},
		{"BracketThenDot", `$record["test1"].test2`, []string{"$record", "test1", "test2"}, false},
		{"BracketThenBracket", `$record["test1"]["test2"]`, []string{"$record", "test1", "test2"}, false},
		{"DotThenBracket", `$record.test1["test2"]`, []string{"$record", "test1", "test2"}, false},
		{"DotsInBrackets", `$record["test1.test2"]`, []string{"$record", "test1.test2"}, false},
		{"UnclosedBrackets", `$record["test1.test2"`, nil, true},
		{"UnclosedQuotes", `$record["test1.test2]`, nil, true},
		{"UnmatchedQuotes", `$record["test1.test2']`, nil, true},
		{"BracketAtEnd", `$record[`, nil, true},
		{"SingleQuoteAtEnd", `$record['`, nil, true},
		{"DoubleQuoteAtEnd", `$record["`, nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := splitField(tc.input)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tc.output, s)
		})
	}
}
