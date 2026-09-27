package settings

import "strings"

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
		if !IsOpenCodeChatModel(s.LLMModel) {
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

// OpenCodeChatModels are Zen models that speak OpenAI chat/completions (Kaiban's adapter).
func OpenCodeChatModels() []string {
	return []string{
		"space-bunny-free",
		"longcat-2.5-preview-free",
		"mimo-v2.6-flash-free",
		"mimo-v2.5-free",
		"ling-3.0-flash-fin-free",
		"nemotron-3-ultra-free",
		"nemotron-3.5-lightning-free",
		"deepseek-v4-flash-free",
		"big-pickle",
		"glm-5.3-flash",
		"glm-5.3",
		"glm-5.2",
		"glm-5.1",
		"glm-5",
		"deepseek-v4.1-flash",
		"deepseek-v4-pro",
		"deepseek-v4-flash",
		"deepseek-v4-flash-vision-exp",
		"minimax-m3",
		"minimax-m2.7",
		"minimax-m2.5",
		"kimi-k3",
		"kimi-k2.7-code",
		"kimi-k2.6",
		"kimi-k2.5",
		"qwen3.8-max",
	}
}

// IsOpenCodeChatModel reports whether model id is known to use /chat/completions on Zen.
func IsOpenCodeChatModel(model string) bool {
	for _, id := range OpenCodeChatModels() {
		if id == model {
			return true
		}
	}
	return false
}
