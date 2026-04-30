package translator

import (
	"context"
	"testing"
)

type MockLLM struct {
	ShouldFail bool
}

func (m *MockLLM) ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	if m.ShouldFail {
		return nil, ErrTranslationFailed
	}

	content := messages[0].Content

	if contains(content, "Translate the following subtitles") {
		return &ChatCompletionResponse{
			Content: `["哈囉大家好","我是布魯諾","這是測試"]`,
		}, nil
	}

	if contains(content, "Review these direct translations") {
		return &ChatCompletionResponse{
			Content: `["","OK","Too long"]`,
		}, nil
	}

	if contains(content, "Create final polished translations") {
		return &ChatCompletionResponse{
			Content: `["哈囉大家好","我是布魯諾","這是測試影片"]`,
		}, nil
	}

	return &ChatCompletionResponse{
		Content: `["Translated"]`,
	}, nil
}

func (m *MockLLM) Name() string {
	return "mock-llm"
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

var ErrTranslationFailed = &TranslationError{Message: "translation failed"}

type TranslationError struct {
	Message string
}

func (e *TranslationError) Error() string {
	return e.Message
}

func TestReflectiveTranslator_DirectTranslate(t *testing.T) {
	llm := &MockLLM{}
	translator := NewReflectiveTranslator(llm)

	chunk := &Chunk{
		Index:      0,
		SourceLang: "en",
		TargetLang: "繁體中文",
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello everyone"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "I am Bruno"},
			{Index: 2, Start: 4.0, End: 6.0, Original: "This is a test"},
		},
	}

	err := translator.DirectTranslate(context.Background(), chunk)
	if err != nil {
		t.Fatalf("DirectTranslate failed: %v", err)
	}

	if chunk.Segments[0].Direct != "哈囉大家好" {
		t.Errorf("expected '哈囉大家好', got '%s'", chunk.Segments[0].Direct)
	}
	if chunk.Segments[1].Direct != "我是布魯諾" {
		t.Errorf("expected '我是布魯諾', got '%s'", chunk.Segments[1].Direct)
	}
}

func TestReflectiveTranslator_ReflectTranslate(t *testing.T) {
	llm := &MockLLM{}
	translator := NewReflectiveTranslator(llm)

	chunk := &Chunk{
		Index:      0,
		SourceLang: "en",
		TargetLang: "繁體中文",
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello", Direct: "哈囉"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "World", Direct: "世界"},
			{Index: 2, Start: 4.0, End: 6.0, Original: "Test", Direct: "這是一個非常長的測試句子"},
		},
	}

	err := translator.ReflectTranslate(context.Background(), chunk)
	if err != nil {
		t.Fatalf("ReflectTranslate failed: %v", err)
	}

	if chunk.Segments[0].Reflection != "" {
		t.Errorf("expected empty reflection for segment 0, got '%s'", chunk.Segments[0].Reflection)
	}
	if chunk.Segments[2].Reflection != "Too long" {
		t.Errorf("expected 'Too long' for segment 2, got '%s'", chunk.Segments[2].Reflection)
	}
}

func TestReflectiveTranslator_FinalTranslate(t *testing.T) {
	llm := &MockLLM{}
	translator := NewReflectiveTranslator(llm)

	chunk := &Chunk{
		Index:      0,
		SourceLang: "en",
		TargetLang: "繁體中文",
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello", Direct: "哈囉", Reflection: ""},
			{Index: 1, Start: 2.0, End: 4.0, Original: "World", Direct: "世界", Reflection: "OK"},
			{Index: 2, Start: 4.0, End: 6.0, Original: "Test", Direct: "這是一個非常長的測試句子", Reflection: "Too long"},
		},
	}

	err := translator.FinalTranslate(context.Background(), chunk)
	if err != nil {
		t.Fatalf("FinalTranslate failed: %v", err)
	}

	if chunk.Segments[0].Final != "哈囉大家好" {
		t.Errorf("expected '哈囉大家好', got '%s'", chunk.Segments[0].Final)
	}
	if chunk.Segments[2].Final != "這是測試影片" {
		t.Errorf("expected '這是測試影片', got '%s'", chunk.Segments[2].Final)
	}
}

func TestReflectiveTranslator_TranslateChunk(t *testing.T) {
	llm := &MockLLM{}
	translator := NewReflectiveTranslator(llm)

	chunk := &Chunk{
		Index:      0,
		SourceLang: "en",
		TargetLang: "繁體中文",
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello everyone"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "I am Bruno"},
			{Index: 2, Start: 4.0, End: 6.0, Original: "This is a test"},
		},
	}

	err := translator.TranslateChunk(context.Background(), chunk)
	if err != nil {
		t.Fatalf("TranslateChunk failed: %v", err)
	}

	if chunk.Segments[0].Direct == "" {
		t.Error("Direct should be set")
	}
	if chunk.Segments[0].Final == "" {
		t.Error("Final should be set")
	}
}

func TestReflectiveTranslator_LLMError(t *testing.T) {
	llm := &MockLLM{ShouldFail: true}
	translator := NewReflectiveTranslator(llm)

	chunk := &Chunk{
		Index:      0,
		SourceLang: "en",
		TargetLang: "繁體中文",
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello"},
		},
	}

	err := translator.DirectTranslate(context.Background(), chunk)
	if err == nil {
		t.Error("expected error when LLM fails")
	}
}

func TestParseJSONArray(t *testing.T) {
	translator := &ReflectiveTranslator{}

	t.Run("simple array", func(t *testing.T) {
		result, err := translator.parseJSONArray(`["a","b","c"]`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 elements, got %d", len(result))
		}
	})

	t.Run("with markdown", func(t *testing.T) {
		result, err := translator.parseJSONArray("```json\n[\"a\",\"b\"]\n```")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 elements, got %d", len(result))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := translator.parseJSONArray("not json")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestTranslateAll(t *testing.T) {
	llm := &MockLLM{}
	translator := NewReflectiveTranslator(llm)
	chunker := NewChunker(DefaultChunkerConfig())

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "World"},
			{Index: 2, Start: 4.0, End: 6.0, Original: "Test"},
		},
		Language: "en",
	}

	chunks := chunker.Split(transcript)

	for _, chunk := range chunks {
		err := translator.TranslateChunk(context.Background(), chunk)
		if err != nil {
			t.Fatalf("TranslateChunk failed: %v", err)
		}
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
	for _, chunk := range chunks {
		for _, seg := range chunk.Segments {
			if seg.Final == "" {
				t.Errorf("segment %d Final is empty", seg.Index)
			}
		}
	}
}