package column

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OutputFieldType string

const (
	OutputString   OutputFieldType = "string"
	OutputURL      OutputFieldType = "url"
	OutputMarkdown OutputFieldType = "markdown"
)

// OutputField is a required/optional result key for a column stage contract.
type OutputField struct {
	Key      string          `json:"key"`
	Label    string          `json:"label"`
	Required bool            `json:"required"`
	Type     OutputFieldType `json:"type"`
}

// BudgetLimits are optional per-column overrides (nil = inherit).
type BudgetLimits struct {
	MaxTokens    *int     `json:"max_tokens,omitempty"`
	MaxCostUSD   *float64 `json:"max_cost_usd,omitempty"`
	MaxWallSec   *int     `json:"max_wall_sec,omitempty"`
	MaxToolCalls *int     `json:"max_tool_calls,omitempty"`
	MaxLLMSteps  *int     `json:"max_llm_steps,omitempty"`
}

type Column struct {
	ID                   uuid.UUID         `json:"id"`
	Name                 string            `json:"name"`
	NameI18n             map[string]string `json:"name_i18n"`
	SystemPromptDefault  string            `json:"system_prompt_default"`
	SystemPromptTemplate string            `json:"system_prompt_template"`
	UserCustomPrompt     *string           `json:"user_custom_prompt"`
	OutputFields         []OutputField     `json:"output_fields"`
	Budget               BudgetLimits      `json:"budget"`
	RequiresGitDiff      bool              `json:"requires_git_diff"`
	OrderIndex           int               `json:"order_index"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

func (c *Column) BuildSystemPrompt() string {
	base := c.SystemPromptTemplate
	if c.UserCustomPrompt == nil || strings.TrimSpace(*c.UserCustomPrompt) == "" {
		return base
	}
	return base + "\n\n# User overlay\n" + *c.UserCustomPrompt
}

func (c *Column) ResetOverlay() {
	c.UserCustomPrompt = nil
	c.UpdatedAt = time.Now().UTC()
}

func (c *Column) RestoreDefaultPrompt() {
	c.SystemPromptTemplate = c.SystemPromptDefault
	c.UpdatedAt = time.Now().UTC()
}

func ValidateOutputs(fields []OutputField, values map[string]string) error {
	if values == nil {
		values = map[string]string{}
	}
	var missing []string
	for _, f := range fields {
		key := strings.TrimSpace(f.Key)
		if key == "" {
			continue
		}
		v := strings.TrimSpace(values[key])
		if f.Required && v == "" {
			missing = append(missing, key)
			continue
		}
		if v == "" {
			continue
		}
		switch f.Type {
		case OutputURL:
			u, err := url.Parse(v)
			if err != nil || u.Scheme == "" || u.Host == "" {
				return fmt.Errorf("output %q must be a valid URL", key)
			}
		case OutputString, OutputMarkdown, "":
			// ok
		default:
			return fmt.Errorf("output %q has unknown type %q", key, f.Type)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required outputs: %s", strings.Join(missing, ", "))
	}
	return nil
}

func NormalizeOutputFields(fields []OutputField) []OutputField {
	out := make([]OutputField, 0, len(fields))
	seen := map[string]struct{}{}
	for _, f := range fields {
		key := strings.TrimSpace(f.Key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		label := strings.TrimSpace(f.Label)
		if label == "" {
			label = key
		}
		typ := f.Type
		switch typ {
		case OutputURL, OutputMarkdown:
		default:
			typ = OutputString
		}
		out = append(out, OutputField{Key: key, Label: label, Required: f.Required, Type: typ})
	}
	return out
}
