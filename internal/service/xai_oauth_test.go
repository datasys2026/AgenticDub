package service

import (
	"errors"
	"path/filepath"
	"testing"

	"krillin-ai/config"
	xaiauth "krillin-ai/internal/auth/xai"
	xaistt "krillin-ai/internal/providers/stt"
	xaitts "krillin-ai/internal/providers/tts"
	"krillin-ai/log"
)

func TestNewServiceWithConfig_XAIOAuthLLM(t *testing.T) {
	log.InitLogger()

	conf := config.Conf
	conf.Llm.Provider = "xai-oauth"
	conf.Llm.BaseURL = "https://api.x.ai/v1"
	conf.Llm.Model = "grok-4.20-0309-non-reasoning"
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
	conf.Llm.Model = "grok-4.20-0309-non-reasoning"
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

func TestNewServiceWithConfig_XAIOAuthSTTAndTTSWithToken(t *testing.T) {
	log.InitLogger()

	tokenPath := filepath.Join(t.TempDir(), "xai-token.json")
	if err := xaiauth.NewFileTokenStore(tokenPath).Save(xaiauth.Token{AccessToken: "oauth-token"}); err != nil {
		t.Fatalf("save token: %v", err)
	}

	conf := config.Conf
	conf.XAI.TokenPath = tokenPath
	conf.XAI.BaseURL = "https://api.x.ai/v1"
	conf.Llm.Provider = "aiark"
	conf.Transcribe.Provider = "xai-oauth"
	conf.Transcribe.Openai.BaseUrl = "https://api.x.ai/v1"
	conf.Tts.Provider = "xai-oauth"
	conf.Tts.Openai.BaseUrl = "https://api.x.ai/v1"

	svc, err := NewServiceWithConfig(conf)
	if err != nil {
		t.Fatalf("NewServiceWithConfig failed: %v", err)
	}
	if _, ok := svc.Transcriber.(*xaistt.XAIOAuthProvider); !ok {
		t.Fatalf("expected xAI STT provider, got %T", svc.Transcriber)
	}
	if _, ok := svc.TtsClient.(*xaitts.XAIOAuthClient); !ok {
		t.Fatalf("expected xAI TTS client, got %T", svc.TtsClient)
	}
}
