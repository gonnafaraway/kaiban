package settings

import (
	"strings"

	"kaiban/internal/api/llmprovider"
)

const (
	ProviderOpenAI   = "openai"
	ProviderOpenCode = "opencode"
	ProviderCustom   = "custom"

	OpenCodeBaseURL      = "https://opencode.ai/zen/v1"
	OpenCodeDefaultModel = "space-bunny-free"
	OpenAIBaseURL        = "https://api.openai.com/v1"
	OpenAIDefaultModel   = "gpt-4.1"
)

// NormalizeProvider returns a known provider id.
func NormalizeProvider(p string) string {
	switch p {
	case ProviderOpenCode, "opencode-zen", "zen":
		return ProviderOpenCode
	case ProviderCustom:
		return ProviderCustom
	case ProviderOpenAI, "":
		return ProviderOpenAI
	default:
		return ProviderCustom
	}
}

// InferProviderFromURL guesses provider from base URL.
func InferProviderFromURL(baseURL string) string {
	switch {
	case strings.Contains(baseURL, "opencode.ai/zen"):
		return ProviderOpenCode
	case baseURL == "" || strings.Contains(baseURL, "api.openai.com"):
		return ProviderOpenAI
	default:
		return ProviderCustom
	}
}

// ApplyProvider sets provider and fills base URL / default model when switching presets.
func ApplyProvider(s *Settings, provider string) {
	p := NormalizeProvider(provider)
	s.LLMProvider = p
	switch p {
	case ProviderOpenCode:
		s.LLMBaseURL = OpenCodeBaseURL
		if !llmprovider.IsOpenCodeChatModel(s.LLMModel) {
			s.LLMModel = OpenCodeDefaultModel
		}
	case ProviderOpenAI:
		if s.LLMBaseURL == "" || InferProviderFromURL(s.LLMBaseURL) != ProviderOpenAI {
			s.LLMBaseURL = OpenAIBaseURL
		}
		if s.LLMModel == "" {
			s.LLMModel = OpenAIDefaultModel
		}
	case ProviderCustom:
		// keep URL/model as provided
	}
}
