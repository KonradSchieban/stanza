package parser

import (
	"context"
	"testing"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	jsoniter "github.com/json-iterator/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func NewFakeJSONPlugin() (*JSONParser, *testutil.Plugin) {
	mock := testutil.Plugin{}
	logger, _ := zap.NewProduction()
	return &JSONParser{
		ParserPlugin: helper.ParserPlugin{
			BasicPlugin: helper.BasicPlugin{
				PluginID:      "test",
				PluginType:    "json_parser",
				SugaredLogger: logger.Sugar(),
			},
			Output:    &mock,
			ParseFrom: entry.NewField("testfield"),
			ParseTo:   entry.NewField("testparsed"),
		},
		json: jsoniter.ConfigFastest,
	}, &mock
}

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(JSONParser))
}

func TestJSONParser(t *testing.T) {
	cases := []struct {
		name           string
		inputRecord    map[string]interface{}
		expectedRecord map[string]interface{}
		errorExpected  bool
	}{
		{
			"simple",
			map[string]interface{}{
				"testfield": `{}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{},
			},
			false,
		},
		{
			"nested",
			map[string]interface{}{
				"testfield": `{"superkey":"superval"}`,
			},
			map[string]interface{}{
				"testparsed": map[string]interface{}{
					"superkey": "superval",
				},
			},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := entry.New()
			input.Record = tc.inputRecord

			output := entry.New()
			output.Record = tc.expectedRecord

			parser, mockOutput := NewFakeJSONPlugin()
			mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				e := args[1].(*entry.Entry)
				if !assert.Equal(t, tc.expectedRecord, e.Record) {
					t.FailNow()
				}
			}).Return(nil)

			err := parser.Process(context.Background(), input)
			if !assert.NoError(t, err) {
				return
			}
		})
	}
}