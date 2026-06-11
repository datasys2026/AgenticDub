package service

import (
	"errors"
	"path/filepath"
	"testing"

	"krillin-ai/config"
	xaiauth "krillin-ai/internal/auth/xai"
	"krillin-ai/log"
)

func TestNewServiceWithConfig_XAIOAuthLLM(t *testing.T) {
	log.InitLogger()

	conf := config.Conf
	conf.Llm.Provider = "xai-oauth"
	conf.Llm.BaseURL = "https://api.x.ai/v1"
	conf.Llm.Model = "grok-4.3"
	conf.XAI.TokenPath = filepath.Join(t.TempDir(), "missing-xai-token.json")

	svc, err := NewServiceWithConfig(conf)
	if svc != nil {
		t.Fatal("expected no service when xAI token is missing")
	}
	if !errors.Is(err, xaiauth.ErrTokenNotFound) {
		t.Fatalf("expected missing xAI token error, got %v", err)
	}
}

func TestNewServiceWithConfig_XAIOAuthLLMWithToken(t *testing.T) {
	log.InitLogger()

	tokenPath := filepath.Join(t.TempDir(), "xai-token.json")
	if err := xaiauth.NewFileTokenStore(tokenPath).Save(xaiauth.Token{AccessToken: "oauth-token"}); err != nil {
		t.Fatalf("save token: %v", err)
	}

	conf := config.Conf
	conf.Llm.Provider = "xai-oauth"
	conf.Llm.BaseURL = "https://api.x.ai/v1"
	conf.Llm.Model = "grok-4.3"
	conf.XAI.TokenPath = tokenPath
	conf.Transcribe.Provider = "openai"
	conf.Tts.Provider = "openai"

	svc, err := NewServiceWithConfig(conf)
	if err != nil {
		t.Fatalf("NewServiceWithConfig failed: %v", err)
	}
	if svc == nil {
		t.Fatal("expected service")
	}
	if svc.ChatCompleter == nil {
		t.Fatal("expected ChatCompleter")
	}
}
