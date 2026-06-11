package stt

import (
	"context"
	"encoding/json"
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

func TestBuildXAISTTURL(t *testing.T) {
	tests := []struct {
		baseURL  string
		expected string
	}{
		{"https://api.x.ai", "https://api.x.ai/v1/stt"},
		{"https://api.x.ai/v1", "https://api.x.ai/v1/stt"},
		{"https://api.x.ai/v1/", "https://api.x.ai/v1/stt"},
	}

	for _, tt := range tests {
		if got := buildXAISTTURL(tt.baseURL); got != tt.expected {
			t.Fatalf("expected %q, got %q", tt.expected, got)
		}
	}
}

func TestXAIOAuthProviderTranscription(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "audio.mp3")
	if err := os.WriteFile(audioPath, []byte("fake audio"), 0644); err != nil {
		t.Fatalf("write audio: %v", err)
	}

	var capturedAuth string
	var capturedLanguage string
	var capturedFormat string
	var capturedFilename string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/stt" {
			t.Fatalf("expected /v1/stt, got %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("ParseMultipartForm failed: %v", err)
		}
		capturedLanguage = r.FormValue("language")
		capturedFormat = r.FormValue("format")
		fileHeaders := r.MultipartForm.File["file"]
		if len(fileHeaders) != 1 {
			t.Fatalf("expected one file, got %d", len(fileHeaders))
		}
		capturedFilename = fileHeaders[0].Filename

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text":     "Hello world",
			"language": "en",
			"duration": 1.4,
			"words": []map[string]any{
				{"text": "Hello", "start": 0.1, "end": 0.5},
				{"text": "world", "start": 0.7, "end": 1.2},
			},
		})
	}))
	defer server.Close()

	provider := NewXAIOAuthProvider(server.URL, staticBearerTokenSource{token: "oauth-token"}, server.Client())
	result, err := provider.Transcription(audioPath, "en", t.TempDir())
	if err != nil {
		t.Fatalf("Transcription failed: %v", err)
	}
	if capturedAuth != "Bearer oauth-token" {
		t.Fatalf("expected bearer token, got %q", capturedAuth)
	}
	if capturedLanguage != "en" {
		t.Fatalf("expected language en, got %q", capturedLanguage)
	}
	if capturedFormat != "true" {
		t.Fatalf("expected format true, got %q", capturedFormat)
	}
	if capturedFilename != "audio.mp3" {
		t.Fatalf("expected audio.mp3 filename, got %q", capturedFilename)
	}
	if result.Text != "Hello world" {
		t.Fatalf("expected transcript text, got %q", result.Text)
	}
	if len(result.Words) != 2 || result.Words[0].Text != "Hello" {
		t.Fatalf("unexpected words: %#v", result.Words)
	}
	if len(result.Segments) != 1 || result.Segments[0].End != 1.4 {
		t.Fatalf("unexpected segments: %#v", result.Segments)
	}
}
