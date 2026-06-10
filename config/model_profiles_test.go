package config

import "testing"

func TestConfigForModelProfiles_XAIOAuthLLM(t *testing.T) {
	base := Conf
	base.Models.LLM = map[string]ModelProfileConfig{
		"grok": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "grok-4.3",
		},
	}

	resolved, err := ConfigForModelProfiles(base, "grok", "", "", "")
	if err != nil {
		t.Fatalf("ConfigForModelProfiles failed: %v", err)
	}
	if resolved.Llm.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth provider, got %q", resolved.Llm.Provider)
	}
	if resolved.Llm.BaseURL != "https://api.x.ai/v1" {
		t.Fatalf("expected xAI base URL, got %q", resolved.Llm.BaseURL)
	}
	if resolved.Llm.Model != "grok-4.3" {
		t.Fatalf("expected grok-4.3 model, got %q", resolved.Llm.Model)
	}
}
