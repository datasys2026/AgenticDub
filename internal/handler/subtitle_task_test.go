package handler

import (
	"errors"
	"path/filepath"
	"testing"

	"krillin-ai/config"
	xaiauth "krillin-ai/internal/auth/xai"
	"krillin-ai/internal/dto"
	"krillin-ai/log"
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
			Model:    "grok-4.20-0309-non-reasoning",
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
	if resolved.Llm.Model != "grok-4.20-0309-non-reasoning" {
		t.Fatalf("expected grok-4.20-0309-non-reasoning model, got %q", resolved.Llm.Model)
	}
}

func TestConfigForSubtitleTaskAppliesXAISTTAndTTSProfiles(t *testing.T) {
	originalConf := config.Conf
	t.Cleanup(func() {
		config.Conf = originalConf
	})

	config.Conf.Models.STT = map[string]config.ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "xai-stt",
		},
	}
	config.Conf.Models.TTS = map[string]config.ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "xai-tts",
			Voices:   []string{"eve", "ara", "rex", "sal", "leo"},
		},
	}

	resolved, ok, err := configForSubtitleTask(dto.StartVideoSubtitleTaskReq{
		STTProfile:   "xai",
		TTSProfile:   "xai",
		TtsVoiceCode: "eve",
	})
	if err != nil {
		t.Fatalf("configForSubtitleTask failed: %v", err)
	}
	if !ok {
		t.Fatal("expected profile-specific config")
	}
	if resolved.Transcribe.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth STT provider, got %q", resolved.Transcribe.Provider)
	}
	if resolved.Tts.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth TTS provider, got %q", resolved.Tts.Provider)
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

func TestServiceForSubtitleTaskReturnsProfileServiceInitError(t *testing.T) {
	log.InitLogger()
	originalConf := config.Conf
	t.Cleanup(func() {
		config.Conf = originalConf
	})

	config.Conf.Transcribe.Provider = "openai"
	config.Conf.Tts.Provider = "openai"
	config.Conf.XAI.TokenPath = filepath.Join(t.TempDir(), "missing-xai-token.json")
	config.Conf.Models.LLM = map[string]config.ModelProfileConfig{
		"grok": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "grok-4.20-0309-non-reasoning",
		},
	}

	svc, err := serviceForSubtitleTask(nil, dto.StartVideoSubtitleTaskReq{
		LLMProfile: "grok",
	})
	if svc != nil {
		t.Fatal("expected no service")
	}
	if !errors.Is(err, xaiauth.ErrTokenNotFound) {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}
