package file

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"golang.org/x/text/encoding"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/helper"
)

func init() {
	operator.Register("file_input", func() operator.Builder { return NewInputConfig("") })
}

const (
	defaultMaxLogSize         = 1024 * 1024
	defaultMaxConcurrentFiles = 1024
)

// NewInputConfig creates a new input config with default values
func NewInputConfig(operatorID string) *InputConfig {
	return &InputConfig{
		InputConfig:        helper.NewInputConfig(operatorID, "file_input"),
		DecoderConfig:      helper.NewDecoderConfig("nop"),
		PollInterval:       helper.Duration{Duration: 200 * time.Millisecond},
		IncludeFileName:    true,
		IncludeFilePath:    false,
		StartAt:            "end",
		MaxLogSize:         defaultMaxLogSize,
		MaxConcurrentFiles: defaultMaxConcurrentFiles,
	}
}

// InputConfig is the configuration of a file input operator
type InputConfig struct {
	helper.InputConfig   `yaml:",inline"`
	helper.DecoderConfig `yaml:",inline,omitempty"`

	Include []string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`

	PollInterval       helper.Duration  `json:"poll_interval,omitempty"        yaml:"poll_interval,omitempty"`
	Multiline          *MultilineConfig `json:"multiline,omitempty"            yaml:"multiline,omitempty"`
	IncludeFileName    bool             `json:"include_file_name,omitempty"    yaml:"include_file_name,omitempty"`
	IncludeFilePath    bool             `json:"include_file_path,omitempty"    yaml:"include_file_path,omitempty"`
	StartAt            string           `json:"start_at,omitempty"             yaml:"start_at,omitempty"`
	MaxLogSize         helper.ByteSize  `json:"max_log_size,omitempty"         yaml:"max_log_size,omitempty"`
	MaxConcurrentFiles int              `json:"max_concurrent_files,omitempty" yaml:"max_concurrent_files,omitempty"`
}

// MultilineConfig is the configuration a multiline operation
type MultilineConfig struct {
	LineStartPattern string `json:"line_start_pattern" yaml:"line_start_pattern"`
	LineEndPattern   string `json:"line_end_pattern"   yaml:"line_end_pattern"`
}

// Build will build a file input operator from the supplied configuration
func (c InputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if len(c.Include) == 0 {
		return nil, fmt.Errorf("required argument `include` is empty")
	}

	// Ensure includes can be parsed as globs
	for _, include := range c.Include {
		_, err := filepath.Match(include, "matchstring")
		if err != nil {
			return nil, fmt.Errorf("parse include glob: %s", err)
		}
	}

	// Ensure excludes can be parsed as globs
	for _, exclude := range c.Exclude {
		_, err := filepath.Match(exclude, "matchstring")
		if err != nil {
			return nil, fmt.Errorf("parse exclude glob: %s", err)
		}
	}

	if c.MaxLogSize <= 0 {
		return nil, fmt.Errorf("`max_log_size` must be positive")
	}

	if c.MaxConcurrentFiles <= 0 {
		return nil, fmt.Errorf("`max_concurrent_files` must be positive")
	}

	decoder, err := c.DecoderConfig.Build()
	if err != nil {
		return nil, err
	}

	splitFunc, err := c.getSplitFunc(decoder.Encoding)
	if err != nil {
		return nil, err
	}

	var startAtBeginning bool
	switch c.StartAt {
	case "beginning":
		startAtBeginning = true
	case "end":
		startAtBeginning = false
	default:
		return nil, fmt.Errorf("invalid start_at location '%s'", c.StartAt)
	}

	fileNameField := entry.NewNilField()
	if c.IncludeFileName {
		fileNameField = entry.NewLabelField("file_name")
	}

	filePathField := entry.NewNilField()
	if c.IncludeFilePath {
		filePathField = entry.NewLabelField("file_path")
	}

	op := &InputOperator{
		InputOperator:      inputOperator,
		decoder:            decoder,
		Include:            c.Include,
		Exclude:            c.Exclude,
		SplitFunc:          splitFunc,
		PollInterval:       c.PollInterval.Raw(),
		persist:            helper.NewScopedDBPersister(context.Database, c.ID()),
		FilePathField:      filePathField,
		FileNameField:      fileNameField,
		fingerprintBytes:   1000,
		startAtBeginning:   startAtBeginning,
		queuedMatches:      make([]string, 0),
		firstCheck:         true,
		cancel:             func() {},
		knownFiles:         make([]*Reader, 0, 10),
		MaxLogSize:         int(c.MaxLogSize),
		MaxConcurrentFiles: c.MaxConcurrentFiles,
		SeenPaths:          make(map[string]struct{}, 100),
	}

	return []operator.Operator{op}, nil
}

// getSplitFunc will return the split function associated the configured mode.
func (c InputConfig) getSplitFunc(encoding encoding.Encoding) (bufio.SplitFunc, error) {
	if c.Multiline == nil {
		return NewNewlineSplitFunc(encoding)
	}
	endPattern := c.Multiline.LineEndPattern
	startPattern := c.Multiline.LineStartPattern

	switch {
	case endPattern != "" && startPattern != "":
		return nil, fmt.Errorf("only one of line_start_pattern or line_end_pattern can be set")
	case endPattern == "" && startPattern == "":
		return nil, fmt.Errorf("one of line_start_pattern or line_end_pattern must be set")
	case endPattern != "":
		re, err := regexp.Compile("(?m)" + c.Multiline.LineEndPattern)
		if err != nil {
			return nil, fmt.Errorf("compile line end regex: %s", err)
		}
		return NewLineEndSplitFunc(re), nil
	case startPattern != "":
		re, err := regexp.Compile("(?m)" + c.Multiline.LineStartPattern)
		if err != nil {
			return nil, fmt.Errorf("compile line start regex: %s", err)
		}
		return NewLineStartSplitFunc(re), nil
	default:
		return nil, fmt.Errorf("unreachable")
	}
}
