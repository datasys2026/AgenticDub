package handler

import (
	"testing"

	"krillin-ai/config"
	"krillin-ai/internal/dto"
)

func TestConfigForSubtitleTaskAppliesGrokProfile(t *testing.T) {
	originalConf := config.Conf
	t.Cleanup(func() {
		config.Conf = originalConf
	})

	config.Conf.Models.LLM = map[string]config.ModelProfileConfig{
		"grok": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "grok-4.3",
		},
	}

	resolved, ok, err := configForSubtitleTask(dto.StartVideoSubtitleTaskReq{
		LLMProfile: "grok",
	})
	if err != nil {
		t.Fatalf("configForSubtitleTask failed: %v", err)
	}
	if !ok {
		t.Fatal("expected profile-specific config")
	}
	if resolved.Llm.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth provider, got %q", resolved.Llm.Provider)
	}
	if resolved.Llm.Model != "grok-4.3" {
		t.Fatalf("expected grok-4.3 model, got %q", resolved.Llm.Model)
	}
}

func TestConfigForSubtitleTaskNoProfiles(t *testing.T) {
	_, ok, err := configForSubtitleTask(dto.StartVideoSubtitleTaskReq{})
	if err != nil {
		t.Fatalf("configForSubtitleTask failed: %v", err)
	}
	if ok {
		t.Fatal("expected no profile-specific config")
	}
}
