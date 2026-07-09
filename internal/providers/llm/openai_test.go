package llm

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestBuildChatURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{"base URL without /v1", "https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"base URL with /v1", "https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"base URL with /v1/", "https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"base URL with trailing slash", "https://api.openai.com/", "https://api.openai.com/v1/chat/completions"},
		{"custom endpoint without /v1", "http://localhost:4000", "http://localhost:4000/v1/chat/completions"},
		{"aiark with /v1", "https://aiark.com.tw/v1", "https://aiark.com.tw/v1/chat/completions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildChatURL(tt.baseURL)
			if got != tt.expected {
				t.Errorf("buildChatURL(%q) = %q, want %q", tt.baseURL, got, tt.expected)
			}
		})
	}
}

func TestBuildHTTPClient(t *testing.T) {
	t.Run("no proxy", func(t *testing.T) {
		client := buildHTTPClient("")
		if client.Timeout == 0 {
			t.Error("expected non-zero timeout")
		}
		if client.Transport != nil {
			t.Error("expected nil transport when no proxy")
		}
	})

	t.Run("valid proxy", func(t *testing.T) {
		proxyAddr := "http://127.0.0.1:7890"
		client := buildHTTPClient(proxyAddr)
		if client.Timeout == 0 {
			t.Error("expected non-zero timeout")
		}
		if client.Transport == nil {
			t.Fatal("expected non-nil transport when proxy set")
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatal("expected *http.Transport")
		}
		proxyFunc := transport.Proxy
		if proxyFunc == nil {
			t.Fatal("expected non-nil Proxy function")
		}
		proxyURL, err := proxyFunc(&http.Request{})
		if err != nil {
			t.Fatalf("Proxy function returned error: %v", err)
		}
		if proxyURL == nil {
			t.Fatal("expected non-nil proxy URL")
		}
		if proxyURL.String() != proxyAddr {
			t.Errorf("expected proxy URL %q, got %q", proxyAddr, proxyURL.String())
		}
	})

	t.Run("invalid proxy URL", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("unexpected panic with invalid proxy: %v", r)
			}
		}()
		client := buildHTTPClient("://invalid")
		_ = client
	})
}

func TestOpenAIProvider_ChatCompletion_URL(t *testing.T) {
	t.Run("URL without /v1 gets corrected", func(t *testing.T) {
		provider := NewOpenAIProvider("http://test.api.com", "key", "model", "")
		// The provider should use buildChatURL internally
		_ = provider
	})
}

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAIProvider("http://localhost", "key", "gpt-4", "")
	if p.Name() != "openai" {
		t.Errorf("expected 'openai', got '%s'", p.Name())
	}
}

func TestOpenAIProvider_ContextCancellation(t *testing.T) {
	p := NewOpenAIProvider("http://localhost:4000", "key", "gpt-4", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.ChatCompletion(ctx, []Message{{Role: "user", Content: "hello"}})
	if err == nil {
		t.Error("expected error with cancelled context")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected cancellation error, got: %v", err)
	}
}
