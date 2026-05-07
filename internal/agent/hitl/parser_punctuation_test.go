package hitl_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"krillin-ai/internal/agent/hitl"
)

func TestTxtParser_ParseWithPunctuation(t *testing.T) {
	tmpDir := t.TempDir()
	reviewFile := filepath.Join(tmpDir, "review.txt")

	content := `【第 1 句】 00:00:12,000 --> 00:00:15,500
原文：Hello, world! How are you?
字幕：你好，世界！你好嗎？

【第 2 句】 00:00:15,500 --> 00:00:18,200
原文：I'm fine, thank you.
字幕：我很好，謝謝你。

【第 3 句】 00:00:18,200 --> 00:00:21,000
原文：What's this? It's a test.
字幕：這是什麼？這是一個測試。
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

	if doc.Segments[0].Original != "Hello, world! How are you?" {
		t.Errorf("expected original %q, got %q", "Hello, world! How are you?", doc.Segments[0].Original)
	}
	if doc.Segments[0].Edited != "你好，世界！你好嗎？" {
		t.Errorf("expected edited %q, got %q", "你好，世界！你好嗎？", doc.Segments[0].Edited)
	}

	if doc.Segments[2].Original != "What's this? It's a test." {
		t.Errorf("expected original %q, got %q", "What's this? It's a test.", doc.Segments[2].Original)
	}
	if doc.Segments[2].Edited != "這是什麼？這是一個測試。" {
		t.Errorf("expected edited %q, got %q", "這是什麼？這是一個測試。", doc.Segments[2].Edited)
	}
}

func TestTxtParser_GenerateWithPunctuation(t *testing.T) {
	doc := hitl.ReviewDocument{
		TaskID:     "punctuation-test",
		VideoTitle: "Test Video",
		Language:   "繁體中文",
		Segments: []hitl.Segment{
			{
				Index:    1,
				Start:    time.Date(0, 1, 1, 0, 0, 12, 0, time.UTC),
				End:      time.Date(0, 1, 1, 0, 0, 15, 500000000, time.UTC),
				Original: "Hello, world!",
				Edited:   "你好，世界！",
			},
		},
	}

	parser := hitl.TxtParser{}
	content, err := parser.Generate(doc)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := "【第 1 句】 00:00:12,000 --> 00:00:15,500\n原文：Hello, world!\n字幕：你好，世界！\n\n"
	if content != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, content)
	}
}

func TestReviewContent_EditableWithPunctuation(t *testing.T) {
	tmpDir := t.TempDir()

	taskDir := filepath.Join(tmpDir, "task-456")
	os.MkdirAll(taskDir, 0755)

	srtContent := `1
00:00:12,000 --> 00:00:15,500
你好，世界！你好嗎？

2
00:00:15,500 --> 00:00:18,200
這是什麼？這是一個測試。
`
	srtPath := filepath.Join(taskDir, "translated.srt")
	os.WriteFile(srtPath, []byte(srtContent), 0644)

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, _ := svc.CreateReview("task-456", srtPath, "Test Video", "繁體中文")
	reviewPath := filepath.Join(taskDir, "review.txt")
	svc.SaveReview(doc, reviewPath)

	contentBytes, _ := os.ReadFile(reviewPath)
	t.Logf("Generated review.txt:\n%s", string(contentBytes))

	newContent := strings.Replace(string(contentBytes), "字幕：這是什麼？這是一個測試。", "字幕：這是啥？這是測試。", 1)
	os.WriteFile(reviewPath, []byte(newContent), 0644)

	_, err := svc.Approve("task-456", reviewPath)
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	finalPath := filepath.Join(taskDir, "final.srt")
	finalContent, _ := os.ReadFile(finalPath)
	t.Logf("Final SRT:\n%s", string(finalContent))

	finalStr := string(finalContent)
	if strings.Contains(finalStr, "這是啥 這是測試") {
		t.Log("Punctuation cleaned in edited content - SUCCESS")
	} else {
		t.Error("final.srt should contain edited text without punctuation: '這是啥 這是測試'")
	}
}