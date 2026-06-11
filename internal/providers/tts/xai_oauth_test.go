package tts

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type staticBearerTokenSource struct {
	token string
	err   error
}

func (s staticBearerTokenSource) BearerToken(ctx context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.token, nil
}

func TestBuildXAITTSURL(t *testing.T) {
	tests := []struct {
		baseURL  string
		expected string
	}{
		{"https://api.x.ai", "https://api.x.ai/v1/tts"},
		{"https://api.x.ai/v1", "https://api.x.ai/v1/tts"},
		{"https://api.x.ai/v1/", "https://api.x.ai/v1/tts"},
	}

	for _, tt := range tests {
		if got := buildXAITTSURL(tt.baseURL); got != tt.expected {
			t.Fatalf("expected %q, got %q", tt.expected, got)
		}
	}
}

func TestXAIOAuthClientText2Speech(t *testing.T) {
	var capturedAuth string
	var capturedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/tts" {
			t.Fatalf("expected /v1/tts, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("mp3-data"))
	}))
	defer server.Close()

	outputFile := filepath.Join(t.TempDir(), "speech.mp3")
	client := NewXAIOAuthClient(server.URL, staticBearerTokenSource{token: "oauth-token"}, server.Client())
	if err := client.Text2Speech("hello", "", outputFile); err != nil {
		t.Fatalf("Text2Speech failed: %v", err)
	}
	if capturedAuth != "Bearer oauth-token" {
		t.Fatalf("expected bearer token, got %q", capturedAuth)
	}
	if capturedBody["text"] != "hello" {
		t.Fatalf("expected text hello, got %q", capturedBody["text"])
	}
	if capturedBody["voice_id"] != DefaultXAIVoice {
		t.Fatalf("expected default voice %q, got %q", DefaultXAIVoice, capturedBody["voice_id"])
	}
	if capturedBody["language"] != DefaultXAILanguage {
		t.Fatalf("expected default language %q, got %q", DefaultXAILanguage, capturedBody["language"])
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if string(data) != "mp3-data" {
		t.Fatalf("expected written audio data, got %q", string(data))
	}
}

func TestXAIOAuthClientText2SpeechEmptyText(t *testing.T) {
	client := NewXAIOAuthClient("https://api.x.ai/v1", staticBearerTokenSource{token: "oauth-token"}, nil)
	err := client.Text2Speech(" ", DefaultXAIVoice, filepath.Join(t.TempDir(), "speech.mp3"))
	if !errors.Is(err, ErrEmptyText) {
		t.Fatalf("expected ErrEmptyText, got %v", err)
	}
}
