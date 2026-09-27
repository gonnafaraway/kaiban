package settings

import "testing"

func TestNormalizeProvider(t *testing.T) {
	if NormalizeProvider("opencode") != ProviderOpenCode {
		t.Fatal("opencode")
	}
	if NormalizeProvider("zen") != ProviderOpenCode {
		t.Fatal("zen")
	}
	if NormalizeProvider("") != ProviderOpenAI {
		t.Fatal("default openai")
	}
}

func TestApplyProviderOpenCode(t *testing.T) {
	s := &Settings{LLMModel: "gpt-4.1", LLMBaseURL: OpenAIBaseURL}
	ApplyProvider(s, ProviderOpenCode)
	if s.LLMProvider != ProviderOpenCode {
		t.Fatalf("provider %q", s.LLMProvider)
	}
	if s.LLMBaseURL != OpenCodeBaseURL {
		t.Fatalf("url %q", s.LLMBaseURL)
	}
	if s.LLMModel != OpenCodeDefaultModel {
		t.Fatalf("model %q", s.LLMModel)
	}
}

func TestApplyProviderKeepsOpenCodeModel(t *testing.T) {
	s := &Settings{LLMModel: "glm-5.1"}
	ApplyProvider(s, ProviderOpenCode)
	if s.LLMModel != "glm-5.1" {
		t.Fatalf("model %q", s.LLMModel)
	}
}

func TestInferProviderFromURL(t *testing.T) {
	if InferProviderFromURL("https://opencode.ai/zen/v1") != ProviderOpenCode {
		t.Fatal("zen url")
	}
	if InferProviderFromURL("https://api.openai.com/v1") != ProviderOpenAI {
		t.Fatal("openai url")
	}
	if InferProviderFromURL("http://localhost:11434/v1") != ProviderCustom {
		t.Fatal("custom url")
	}
}
