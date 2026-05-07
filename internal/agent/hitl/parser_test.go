package hitl_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"krillin-ai/internal/agent/hitl"
)

func TestTxtParser_Parse(t *testing.T) {
	tmpDir := t.TempDir()
	reviewFile := filepath.Join(tmpDir, "review.txt")

	content := `【第 1 句】 00:00:12,000 --> 00:00:15,500
原文：Hello world, how are you?
字幕：你今天好嗎？

【第 2 句】 00:00:15,500 --> 00:00:18,200
原文：I'm fine, thank you.
字幕：我很好，謝謝。

【第 3 句】 00:00:18,200 --> 00:00:21,000
原文：This is a test.
字幕：這是一個測試。
`

	err := os.WriteFile(reviewFile, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	parser := hitl.TxtParser{}
	doc, err := parser.Parse(reviewFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(doc.Segments) != 3 {
		t.Errorf("expected 3 segments, got %d", len(doc.Segments))
	}

	if doc.Segments[0].Original != "Hello world, how are you?" {
		t.Errorf("expected original %q, got %q", "Hello world, how are you?", doc.Segments[0].Original)
	}

	if doc.Segments[0].Edited != "你今天好嗎？" {
		t.Errorf("expected edited %q, got %q", "你今天好嗎？", doc.Segments[0].Edited)
	}
}

func TestTxtParser_ParseWithEdits(t *testing.T) {
	tmpDir := t.TempDir()
	reviewFile := filepath.Join(tmpDir, "review.txt")

	content := `【第 1 句】 00:00:12,000 --> 00:00:15,500
原文：Hello world
字幕：你好世界

【第 2 句】 00:00:15,500 --> 00:00:18,200
原文：Good morning
字幕：早安
`

	err := os.WriteFile(reviewFile, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	parser := hitl.TxtParser{}
	doc, err := parser.Parse(reviewFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	editedCount := 0
	for _, seg := range doc.Segments {
		if seg.HasEdit() {
			editedCount++
		}
	}

	if editedCount != 2 {
		t.Errorf("expected 2 edits, got %d", editedCount)
	}
}

func TestTxtParser_Generate(t *testing.T) {
	doc := hitl.ReviewDocument{
		TaskID:     "test-123",
		VideoTitle: "Test Video",
		Language:   "繁體中文",
		Segments: []hitl.Segment{
			{
				Index:    1,
				Start:    time.Date(0, 1, 1, 0, 0, 12, 0, time.UTC),
				End:      time.Date(0, 1, 1, 0, 0, 15, 500000000, time.UTC),
				Original: "Hello world",
				Edited:   "你好世界",
			},
		},
	}

	parser := hitl.TxtParser{}
	content, err := parser.Generate(doc)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := `【第 1 句】 00:00:12,000 --> 00:00:15,500
原文：Hello world
字幕：你好世界

`
	if content != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, content)
	}
}