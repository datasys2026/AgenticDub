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
			Model:    "grok-4.20-0309-non-reasoning",
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
	if resolved.Llm.Model != "grok-4.20-0309-non-reasoning" {
		t.Fatalf("expected grok-4.20-0309-non-reasoning model, got %q", resolved.Llm.Model)
	}
}

func TestConfigForModelProfiles_XAIOAuthSTTAndTTS(t *testing.T) {
	base := Conf
	base.Models.STT = map[string]ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "xai-stt",
		},
	}
	base.Models.TTS = map[string]ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "xai-tts",
			Voices:   []string{"eve", "ara", "rex", "sal", "leo"},
		},
	}

	resolved, err := ConfigForModelProfiles(base, "", "xai", "xai", "eve")
	if err != nil {
		t.Fatalf("ConfigForModelProfiles failed: %v", err)
	}
	if resolved.Transcribe.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth STT provider, got %q", resolved.Transcribe.Provider)
	}
	if resolved.Transcribe.Openai.BaseUrl != "https://api.x.ai/v1" {
		t.Fatalf("expected xAI STT base URL, got %q", resolved.Transcribe.Openai.BaseUrl)
	}
	if resolved.Transcribe.Openai.Model != "xai-stt" {
		t.Fatalf("expected xai-stt model, got %q", resolved.Transcribe.Openai.Model)
	}
	if resolved.Tts.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth TTS provider, got %q", resolved.Tts.Provider)
	}
	if resolved.Tts.Openai.BaseUrl != "https://api.x.ai/v1" {
		t.Fatalf("expected xAI TTS base URL, got %q", resolved.Tts.Openai.BaseUrl)
	}
	if resolved.Tts.Openai.Model != "xai-tts" {
		t.Fatalf("expected xai-tts model, got %q", resolved.Tts.Openai.Model)
	}
	if got := resolved.Tts.Voices; len(got) != 5 || got[0] != "eve" || got[4] != "leo" {
		t.Fatalf("expected xAI TTS voices copied into resolved config, got %#v", got)
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

func TestValidateConfigXAIOAuthTranscribeAllowsMissingOpenAIKey(t *testing.T) {
	originalConf := Conf
	t.Cleanup(func() {
		Conf = originalConf
	})

	Conf.Llm.Provider = "aiark"
	Conf.Transcribe.Provider = "xai-oauth"
	Conf.Transcribe.Openai.ApiKey = ""

	if err := validateConfig(); err != nil {
		t.Fatalf("expected xai-oauth transcribe config to pass, got %v", err)
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
