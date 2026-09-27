package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// StageOutputSink receives structured stage outputs from the agent tool.
type StageOutputSink func(values map[string]string) error

type StageOutputField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
}

type StageOutputTool struct {
	Fields []StageOutputField
	Sink   StageOutputSink
}

type stageOutputWrap struct {
	Values map[string]string `json:"values"`
}

func (StageOutputTool) Name() string { return "submit_stage_output" }

func (t StageOutputTool) Description() string {
	keys := make([]string, 0, len(t.Fields))
	for _, f := range t.Fields {
		req := ""
		if f.Required {
			req = " (required)"
		}
		keys = append(keys, f.Key+req)
	}
	return "Submit structured stage outputs for this column. Keys: " + strings.Join(keys, ", ") +
		". Call this before finishing. Args: JSON object with those keys as string values."
}

func (t StageOutputTool) Parameters() map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, f := range t.Fields {
		props[f.Key] = map[string]any{"type": "string", "description": f.Label}
		if f.Required {
			required = append(required, f.Key)
		}
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func (t StageOutputTool) Call(_ context.Context, args string) (string, error) {
	if t.Sink == nil {
		return "", fmt.Errorf("stage output sink is not configured")
	}
	values := map[string]string{}
	var wrap stageOutputWrap
	if err := json.Unmarshal([]byte(args), &wrap); err == nil && wrap.Values != nil {
		values = wrap.Values
	} else if err := json.Unmarshal([]byte(args), &values); err != nil {
		return "", errors.Wrap(err, "decode stage output args")
	}
	if err := t.Sink(values); err != nil {
		return "", errors.Wrap(err, "submit stage output")
	}
	return "stage outputs accepted", nil
}
