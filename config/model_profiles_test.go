package config

import (
	"runtime"
	"strings"
	"testing"
)

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

func TestValidateConfigUnsupportedLLMMessageIncludesXAIOAuth(t *testing.T) {
	originalConf := Conf
	t.Cleanup(func() {
		Conf = originalConf
	})

	Conf.Llm.Provider = "unknown"
	err := validateConfig()
	if err == nil {
		t.Fatal("expected unsupported LLM error")
	}
	if !strings.Contains(err.Error(), "xai-oauth") {
		t.Fatalf("expected xai-oauth in error message, got %q", err.Error())
	}
}

func TestValidateConfigWhispercppNonWindowsFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows validation only")
	}

	originalConf := Conf
	t.Cleanup(func() {
		Conf = originalConf
	})

	Conf.Llm.Provider = "aiark"
	Conf.Transcribe.Provider = "whispercpp"
	Conf.Transcribe.Whispercpp.Model = "large-v2"

	err := validateConfig()
	if err == nil {
		t.Fatal("expected whispercpp non-Windows error")
	}
	if !strings.Contains(err.Error(), "whispercpp") {
		t.Fatalf("expected whispercpp error, got %q", err.Error())
	}
}
