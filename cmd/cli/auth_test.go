package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	xaiauth "krillin-ai/internal/auth/xai"
)

func TestResolveXAITokenPath(t *testing.T) {
	t.Setenv("XAI_OAUTH_TOKEN_PATH", "")
	if got := resolveXAITokenPath("explicit.json"); got != "explicit.json" {
		t.Fatalf("expected explicit token path, got %q", got)
	}

	t.Setenv("XAI_OAUTH_TOKEN_PATH", "env.json")
	if got := resolveXAITokenPath(""); got != "env.json" {
		t.Fatalf("expected env token path, got %q", got)
	}

	t.Setenv("XAI_OAUTH_TOKEN_PATH", "")
	if got := resolveXAITokenPath(""); got != xaiauth.DefaultTokenPath() {
		t.Fatalf("expected default token path, got %q", got)
	}
}

func TestRunXAIProbe(t *testing.T) {
	var capturedAuth string
	var capturedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("expected /v1/responses, got %s", r.URL.Path)
		}
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		capturedModel = body.Model
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"agenticdub-probe-ok"}`))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "xai.json")
	if err := xaiauth.NewFileTokenStore(path).Save(xaiauth.Token{AccessToken: "oauth-token"}); err != nil {
		t.Fatalf("save token: %v", err)
	}

	var stdout bytes.Buffer
	err := runXAIProbe(t.Context(), xaiProbeOptions{
		TokenPath: path,
		BaseURL:   server.URL,
		Model:     "grok-test",
		Client:    server.Client(),
		Stdout:    &stdout,
	})
	if err != nil {
		t.Fatalf("runXAIProbe failed: %v", err)
	}
	if capturedAuth != "Bearer oauth-token" {
		t.Fatalf("expected bearer token, got %q", capturedAuth)
	}
	if capturedModel != "grok-test" {
		t.Fatalf("expected grok-test model, got %q", capturedModel)
	}
	if !strings.Contains(stdout.String(), "xAI OAuth probe: ok") {
		t.Fatalf("expected ok output, got %q", stdout.String())
	}
}

func TestRunXAIProbeEntitlementError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"subscription required"}}`))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "xai.json")
	if err := xaiauth.NewFileTokenStore(path).Save(xaiauth.Token{AccessToken: "oauth-token"}); err != nil {
		t.Fatalf("save token: %v", err)
	}

	var stdout bytes.Buffer
	err := runXAIProbe(t.Context(), xaiProbeOptions{
		TokenPath: path,
		BaseURL:   server.URL,
		Model:     "grok-test",
		Client:    server.Client(),
		Stdout:    &stdout,
	})
	if err == nil {
		t.Fatal("expected entitlement error")
	}
	if !strings.Contains(err.Error(), "subscription") {
		t.Fatalf("expected subscription hint, got %v", err)
	}
}
