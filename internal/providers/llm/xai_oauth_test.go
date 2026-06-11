package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestBuildXAIResponsesURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{"base without v1", "https://api.x.ai", "https://api.x.ai/v1/responses"},
		{"base with v1", "https://api.x.ai/v1", "https://api.x.ai/v1/responses"},
		{"base with trailing slash", "https://api.x.ai/v1/", "https://api.x.ai/v1/responses"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildXAIResponsesURL(tt.baseURL)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestXAIOAuthProvider_ChatCompletion(t *testing.T) {
	var capturedAuth string
	var capturedPath string
	var capturedModel string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedPath = r.URL.Path

		var body struct {
			Model string `json:"model"`
			Input []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		capturedModel = body.Model
		if len(body.Input) != 1 || body.Input[0].Content != "Hello" {
			t.Fatalf("unexpected input: %#v", body.Input)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"你好"}`))
	}))
	defer server.Close()

	provider := NewXAIOAuthProvider(server.URL, "grok-4.3", staticBearerTokenSource{token: "oauth-token"}, server.Client())
	resp, err := provider.ChatCompletion(context.Background(), []Message{{Role: "user", Content: "Hello"}})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if resp.Content != "你好" {
		t.Fatalf("expected translated content, got %q", resp.Content)
	}
	if capturedAuth != "Bearer oauth-token" {
		t.Fatalf("expected OAuth bearer token, got %q", capturedAuth)
	}
	if capturedPath != "/v1/responses" {
		t.Fatalf("expected /v1/responses, got %q", capturedPath)
	}
	if capturedModel != "grok-4.3" {
		t.Fatalf("expected grok-4.3, got %q", capturedModel)
	}
}

func TestXAIOAuthProvider_ParseOutputContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{"content": [{"type": "output_text", "text": "巢狀輸出"}]}
			]
		}`))
	}))
	defer server.Close()

	provider := NewXAIOAuthProvider(server.URL, "grok-4.3", staticBearerTokenSource{token: "oauth-token"}, server.Client())
	resp, err := provider.ChatCompletion(context.Background(), []Message{{Role: "user", Content: "Hello"}})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if resp.Content != "巢狀輸出" {
		t.Fatalf("expected nested output text, got %q", resp.Content)
	}
}

func TestXAIOAuthProvider_ConcatenatesOutputTextFragments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [
				{"content": [{"type": "reasoning_text", "text": "ignore this"}]},
				{"content": [{"type": "output_text", "text": "第一段"}]},
				{"content": [{"type": "output_text", "text": "第二段"}]}
			]
		}`))
	}))
	defer server.Close()

	provider := NewXAIOAuthProvider(server.URL, "grok-4.3", staticBearerTokenSource{token: "oauth-token"}, server.Client())
	resp, err := provider.ChatCompletion(context.Background(), []Message{{Role: "user", Content: "Hello"}})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if resp.Content != "第一段第二段" {
		t.Fatalf("expected concatenated output text, got %q", resp.Content)
	}
}

func TestXAIOAuthProvider_EntitlementError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"The caller does not have permission to execute the specified operation"}}`))
	}))
	defer server.Close()

	provider := NewXAIOAuthProvider(server.URL, "grok-4.3", staticBearerTokenSource{token: "oauth-token"}, server.Client())
	_, err := provider.ChatCompletion(context.Background(), []Message{{Role: "user", Content: "Hello"}})
	var entitlementErr *XAIEntitlementError
	if !errors.As(err, &entitlementErr) {
		t.Fatalf("expected XAIEntitlementError, got %T: %v", err, err)
	}
	if !strings.Contains(entitlementErr.Error(), "subscription") {
		t.Fatalf("expected subscription hint, got %q", entitlementErr.Error())
	}
}

func TestXAIOAuthProvider_TokenSourceError(t *testing.T) {
	expected := errors.New("missing token")
	provider := NewXAIOAuthProvider("https://api.x.ai", "grok-4.3", staticBearerTokenSource{err: expected}, http.DefaultClient)

	_, err := provider.ChatCompletion(context.Background(), []Message{{Role: "user", Content: "Hello"}})
	if !errors.Is(err, expected) {
		t.Fatalf("expected token source error, got %v", err)
	}
}
