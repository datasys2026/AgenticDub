package hitl_test

import (
	"strings"
	"testing"

	"krillin-ai/internal/agent/hitl"
)

func TestCleanPunctuation_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"你好，世界！", "你好 世界"},
		{"這是什麼？這是一個測試。", "這是什麼 這是一個測試"},
		{"大家好，今天很高興！", "大家好 今天很高興"},
		{"Hello, world!", "Hello world"},
		{"What's this? It's a test.", "What s this It s a test"},
		{"你好。", "你好"},
		{"你好", "你好"},
		{"   ", ""},
		{"標點測試：這是測試；哈哈哈", "標點測試 這是測試 哈哈哈"},
	}

	for _, tt := range tests {
		result := hitl.CleanPunctuation(tt.input)
		if result != tt.expected {
			t.Errorf("CleanPunctuation(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCleanPunctuation_NoTrailingSpace(t *testing.T) {
	input := "你好，世界！"
	result := hitl.CleanPunctuation(input)

	if strings.HasSuffix(result, " ") {
		t.Errorf("result should not end with space, got %q", result)
	}
}

func TestCleanPunctuation_MultipleSpaces(t *testing.T) {
	input := "你好，    世界！"
	result := hitl.CleanPunctuation(input)

	if result != "你好 世界" {
		t.Errorf("expected '你好 世界', got %q", result)
	}
}

func TestCleanPunctuation_AllPunctuation(t *testing.T) {
	input := "，。？！、；：\"\"''（）【】《》「」『』!?,."
	result := hitl.CleanPunctuation(input)

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}