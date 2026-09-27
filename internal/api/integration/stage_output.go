package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// StageOutputSink receives structured stage outputs from the agent tool.
type StageOutputSink func(values map[string]string) error

type StageOutputTool struct {
	Fields []struct {
		Key      string `json:"key"`
		Label    string `json:"label"`
		Required bool   `json:"required"`
		Type     string `json:"type"`
	}
	Sink StageOutputSink
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
	_ = json.Unmarshal([]byte(args), &values)
	// Also accept {"values":{...}}
	var wrap struct {
		Values map[string]string `json:"values"`
	}
	if len(values) == 0 {
		_ = json.Unmarshal([]byte(args), &wrap)
		values = wrap.Values
	}
	if err := t.Sink(values); err != nil {
		return "", err
	}
	return "stage outputs accepted", nil
}
