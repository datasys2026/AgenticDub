package hitl_test

import (
	"os"
	"path/filepath"
	"testing"

	"krillin-ai/internal/agent/hitl"
)

func TestSRTMerger_Merge(t *testing.T) {
	tmpDir := t.TempDir()

	originalSRT := filepath.Join(tmpDir, "original.srt")
	srtContent := `1
00:00:12,000 --> 00:00:15,500
Hello world, how are you?

2
00:00:15,500 --> 00:00:18,200
I'm fine, thank you.

3
00:00:18,200 --> 00:00:21,000
This is a test.
`
	err := os.WriteFile(originalSRT, []byte(srtContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	editedSegments := []hitl.Segment{
		{
			Index:    2,
			Original: "I'm fine, thank you.",
			Edited:   "我很好，謝謝。",
		},
	}

	merger := hitl.SRTMerger{}
	result, err := merger.Merge(originalSRT, editedSegments)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if result.Segments[1].Text != "我很好，謝謝。" {
		t.Errorf("expected segment 2 to be edited, got %q", result.Segments[1].Text)
	}

	if result.Segments[0].Text != "Hello world, how are you?" {
		t.Errorf("segment 1 should be unchanged, got %q", result.Segments[0].Text)
	}
}

func TestSRTMerger_MergeMultiple(t *testing.T) {
	tmpDir := t.TempDir()

	originalSRT := filepath.Join(tmpDir, "original.srt")
	srtContent := `1
00:00:00,000 --> 00:00:02,000
Line one

2
00:00:02,000 --> 00:00:04,000
Line two

3
00:00:04,000 --> 00:00:06,000
Line three
`
	err := os.WriteFile(originalSRT, []byte(srtContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	editedSegments := []hitl.Segment{
		{Index: 1, Original: "Line one", Edited: "第一句"},
		{Index: 3, Original: "Line three", Edited: "第三句"},
	}

	merger := hitl.SRTMerger{}
	result, err := merger.Merge(originalSRT, editedSegments)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if len(result.Segments) != 3 {
		t.Errorf("expected 3 segments, got %d", len(result.Segments))
	}

	if result.Segments[0].Text != "第一句" {
		t.Errorf("segment 1 expected '第一句', got %q", result.Segments[0].Text)
	}

	if result.Segments[2].Text != "第三句" {
		t.Errorf("segment 3 expected '第三句', got %q", result.Segments[2].Text)
	}
}

func TestSRTMerger_WriteSRTFile(t *testing.T) {
	tmpDir := t.TempDir()

	merged := hitl.MergedSRT{
		Segments: []hitl.SRTSegment{
			{Index: 1, Start: "00:00:12,000", End: "00:00:15,500", Text: "你好世界"},
			{Index: 2, Start: "00:00:15,500", End: "00:00:18,200", Text: "我很好"},
		},
	}

	outputPath := filepath.Join(tmpDir, "output.srt")
	merger := hitl.SRTMerger{}
	err := merger.Write(outputPath, merged)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	expected := `1
00:00:12,000 --> 00:00:15,500
你好世界

2
00:00:15,500 --> 00:00:18,200
我很好

`
	if string(content) != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, string(content))
	}
}