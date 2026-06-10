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

	svc := NewServiceWithConfig(conf)
	if svc == nil {
		t.Fatal("expected service")
	}
	if svc.ChatCompleter == nil {
		t.Fatal("expected ChatCompleter")
	}

	_, err := svc.ChatCompleter.ChatCompletion("hello")
	if !errors.Is(err, xaiauth.ErrTokenNotFound) {
		t.Fatalf("expected missing xAI token error, got %v", err)
	}
}
