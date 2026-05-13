package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestLLMProviderInterface(t *testing.T) {
	t.Run("LLMProvider must implement ChatCompletion", func(t *testing.T) {
		var provider LLMProvider = &MockLLMProvider{}

		ctx := context.Background()
		resp, err := provider.ChatCompletion(ctx, []Message{
			{Role: "user", Content: "Hello"},
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if resp.Content == "" {
			t.Fatal("expected non-empty content")
		}
	})

	t.Run("LLMProvider must return name", func(t *testing.T) {
		provider := &MockLLMProvider{}
		if provider.Name() == "" {
			t.Error("expected non-empty name")
		}
	})
}

type MockLLMProvider struct{}

func (m *MockLLMProvider) ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	return &ChatCompletionResponse{
		Content: "Mock response: " + messages[0].Content,
	}, nil
}

func (m *MockLLMProvider) Name() string {
	return "mock-llm"
}

func TestDirectTranslation(t *testing.T) {
	provider := &MockLLMProvider{}
	ctx := context.Background()

	prompt := "Translate to 繁體中文: Hello world"
	messages := []Message{{Role: "user", Content: prompt}}

	resp, err := provider.ChatCompletion(ctx, messages)
	if err != nil {
		t.Fatalf("translation failed: %v", err)
	}
	t.Logf("Translated: %s", resp.Content)
}

func TestReflectiveTranslation(t *testing.T) {
	provider := &MockLLMProvider{}
	ctx := context.Background()

	originals := []string{"Hello", "World"}
	translated := []string{"你好", "世界"}

	t.Run("Direct step", func(t *testing.T) {
		prompt := fmt.Sprintf("Direct translate to 繁體中文: %v", originals)
		_, err := provider.ChatCompletion(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			t.Fatalf("direct step failed: %v", err)
		}
	})

	t.Run("Reflect step", func(t *testing.T) {
		prompt := fmt.Sprintf("Review translations: original=%v, direct=%v", originals, translated)
		_, err := provider.ChatCompletion(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			t.Fatalf("reflect step failed: %v", err)
		}
	})

	t.Run("Paraphrase step", func(t *testing.T) {
		prompt := fmt.Sprintf("Final paraphrase with max 42 chars per line")
		_, err := provider.ChatCompletion(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			t.Fatalf("paraphrase step failed: %v", err)
		}
	})
}

func TestTranslationBatch(t *testing.T) {
	provider := &MockLLMProvider{}
	ctx := context.Background()

	segments := []string{
		"This is a test",
		"Hello world",
		"How are you?",
	}

	batchSize := 8
	for i := 0; i < len(segments); i += batchSize {
		end := i + batchSize
		if end > len(segments) {
			end = len(segments)
		}
		batch := segments[i:end]

		prompt := fmt.Sprintf("Translate to 繁體中文: %v", batch)
		_, err := provider.ChatCompletion(ctx, []Message{{Role: "user", Content: prompt}})
		if err != nil {
			t.Fatalf("batch translation failed: %v", err)
		}
		t.Logf("Translated batch %d-%d", i, end)
	}
}

func TestJSONArrayParsing(t *testing.T) {
	t.Run("valid JSON array", func(t *testing.T) {
		jsonStr := `["你好","世界","你好嗎"]`
		var result []string
		err := json.Unmarshal([]byte(jsonStr), &result)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 elements, got %d", len(result))
		}
	})

	t.Run("JSON with surrounding text", func(t *testing.T) {
		jsonStr := `Here is the JSON: ["你好","世界"] and more text`
		trimmed := strings.TrimPrefix(jsonStr, "Here is the JSON: ")
		trimmed = strings.TrimSuffix(trimmed, " and more text")
		var result []string
		err := json.Unmarshal([]byte(trimmed), &result)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 elements, got %d", len(result))
		}
	})
}

func TestAiarkLLMProvider(t *testing.T) {
	t.Run("AiarkLLMProvider implements LLMProvider", func(t *testing.T) {
		provider := NewAiarkLLMProvider("http://localhost:4000", "test-key", "aiark/gemma4-e2b")
		var llmProvider LLMProvider = provider
		if llmProvider.Name() != "aiark-llm" {
			t.Errorf("expected name 'aiark-llm', got '%s'", llmProvider.Name())
		}
	})
}