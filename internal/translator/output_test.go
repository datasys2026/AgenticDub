package translator

import (
	"testing"
)

func TestAligner_Align(t *testing.T) {
	aligner := NewAligner()

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello"},
			{Index: 1, Start: 2.0, End: 4.0, Original: "World"},
		},
		Language: "en",
	}

	translations := []string{"哈囉", "世界"}

	result, err := aligner.Align(transcript, translations)
	if err != nil {
		t.Fatalf("Align failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 segments, got %d", len(result))
	}

	if result[0].Final != "哈囉" {
		t.Errorf("expected '哈囉', got '%s'", result[0].Final)
	}
	if result[0].Start != 0.0 || result[0].End != 2.0 {
		t.Errorf("expected timing 0.0-2.0, got %f-%f", result[0].Start, result[0].End)
	}
}

func TestAligner_Align_CountMismatch(t *testing.T) {
	aligner := NewAligner()

	transcript := &Transcript{
		Segments: []Segment{
			{Index: 0, Start: 0.0, End: 2.0, Original: "Hello"},
		},
		Language: "en",
	}

	translations := []string{"哈囉", "世界"}

	_, err := aligner.Align(transcript, translations)
	if err == nil {
		t.Error("expected error for count mismatch")
	}
}

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		seconds float64
		expected string
	}{
		{0.0, "00:00:00,000"},
		{1.5, "00:00:01,500"},
		{65.0, "00:01:05,000"},
		{3661.0, "01:01:01,000"},
	}

	for _, tt := range tests {
		result := formatTimestamp(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatTimestamp(%f) = %s, expected %s", tt.seconds, result, tt.expected)
		}
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"00:00:00,000", 0.0},
		{"00:00:01,500", 1.5},
		{"00:01:05,000", 65.0},
		{"01:01:01,000", 3661.0},
	}

	for _, tt := range tests {
		result, err := ParseTimestamp(tt.input)
		if err != nil {
			t.Errorf("ParseTimestamp(%s) failed: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("ParseTimestamp(%s) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestSRTGenerator_Generate(t *testing.T) {
	generator := NewSRTGenerator()

	segments := []Segment{
		{Index: 0, Start: 0.0, End: 2.0, Final: "哈囉"},
		{Index: 1, Start: 2.0, End: 4.0, Final: "世界"},
	}

	result := generator.Generate(segments)

	expected := `1
00:00:00,000 --> 00:00:02,000
哈囉

2
00:00:02,000 --> 00:00:04,000
世界

`

	if result != expected {
		t.Errorf("unexpected SRT output:\nGot:\n%s\nExpected:\n%s", result, expected)
	}
}

func TestSRTGenerator_GenerateWithOriginal(t *testing.T) {
	generator := NewSRTGenerator()

	segments := []Segment{
		{Index: 0, Start: 0.0, End: 2.0, Original: "Hello", Final: "哈囉"},
		{Index: 1, Start: 2.0, End: 4.0, Original: "World", Final: "世界"},
	}

	result := generator.GenerateWithOriginal(segments)

	if len(result) == 0 {
		t.Error("expected non-empty SRT")
	}
	if !containsSubstring(result, "Hello") {
		t.Error("expected original text 'Hello' in output")
	}
	if !containsSubstring(result, "哈囉") {
		t.Error("expected translation '哈囉' in output")
	}
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTrimmer_Trim(t *testing.T) {
	trimmer := NewTextTrimmer(nil)

	t.Run("short text", func(t *testing.T) {
		text := "短文字"
		result, err := trimmer.Trim(text, 2.0)
		if err != nil {
			t.Errorf("Trim failed: %v", err)
		}
		if result != text {
			t.Errorf("expected '%s', got '%s'", text, result)
		}
	})

	t.Run("long text with sentence split", func(t *testing.T) {
		text := "這是一個很長的句子. 這是另一個句子"
		result, err := trimmer.Trim(text, 3.0)
		if err != nil {
			t.Errorf("Trim failed: %v", err)
		}
		if len(result) > 50 {
			t.Errorf("expected result <= 50 chars, got %d", len(result))
		}
	})
}

func TestTrimmer_TrimWithZeroDuration(t *testing.T) {
	trimmer := NewTextTrimmer(nil)
	text := "這是一個很長的句子"

	result, err := trimmer.Trim(text, 0)
	if err != nil {
		t.Errorf("Trim failed: %v", err)
	}
	if result != text {
		t.Errorf("expected original text, got '%s'", result)
	}
}