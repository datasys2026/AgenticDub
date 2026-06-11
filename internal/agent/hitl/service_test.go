package hitl_test

import (
	"os"
	"path/filepath"
	"testing"

	"krillin-ai/internal/agent/hitl"
)

func TestReviewService_CreateReview(t *testing.T) {
	tmpDir := t.TempDir()

	srtContent := `1
00:00:12,000 --> 00:00:15,500
Hello world

2
00:00:15,500 --> 00:00:18,200
Good morning
`
	srtPath := filepath.Join(tmpDir, "translated.srt")
	err := os.WriteFile(srtPath, []byte(srtContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, err := svc.CreateReview("task-123", srtPath, "Test Video", "繁體中文")
	if err != nil {
		t.Fatalf("CreateReview failed: %v", err)
	}

	if len(doc.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(doc.Segments))
	}

	if doc.Segments[0].Original != "Hello world" {
		t.Errorf("expected original %q, got %q", "Hello world", doc.Segments[0].Original)
	}
}

func TestReviewService_CreateReviewFromBilingualTargetOnBottom(t *testing.T) {
	tmpDir := t.TempDir()

	srtContent := `1
00:00:00,000 --> 00:00:01,000
Hello world
你好世界

2
00:00:01,000 --> 00:00:02,000
Good morning
早安
`
	srtPath := filepath.Join(tmpDir, "bilingual.srt")
	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		t.Fatal(err)
	}

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, err := svc.CreateReviewFromBilingual("task-123", srtPath, "Test Video", "繁體中文", false)
	if err != nil {
		t.Fatalf("CreateReviewFromBilingual failed: %v", err)
	}
	if got := doc.Segments[0].Original; got != "Hello world" {
		t.Fatalf("expected original English, got %q", got)
	}
	if got := doc.Segments[0].Edited; got != "你好世界" {
		t.Fatalf("expected edited Chinese, got %q", got)
	}
}

func TestReviewService_CreateReviewFromBilingualTargetOnTop(t *testing.T) {
	tmpDir := t.TempDir()

	srtContent := `1
00:00:00,000 --> 00:00:01,000
你好世界
Hello world
`
	srtPath := filepath.Join(tmpDir, "bilingual.srt")
	if err := os.WriteFile(srtPath, []byte(srtContent), 0644); err != nil {
		t.Fatal(err)
	}

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, err := svc.CreateReviewFromBilingual("task-123", srtPath, "Test Video", "繁體中文", true)
	if err != nil {
		t.Fatalf("CreateReviewFromBilingual failed: %v", err)
	}
	if got := doc.Segments[0].Original; got != "Hello world" {
		t.Fatalf("expected original English, got %q", got)
	}
	if got := doc.Segments[0].Edited; got != "你好世界" {
		t.Fatalf("expected edited Chinese, got %q", got)
	}
}

func TestReviewService_Approve(t *testing.T) {
	tmpDir := t.TempDir()

	// Create translated.srt in task directory
	taskDir := filepath.Join(tmpDir, "task-123")
	os.MkdirAll(taskDir, 0755)

	srtContent := `1
00:00:12,000 --> 00:00:15,500
Hello world

2
00:00:15,500 --> 00:00:18,200
Good morning
`
	srtPath := filepath.Join(taskDir, "translated.srt")
	err := os.WriteFile(srtPath, []byte(srtContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	reviewPath := filepath.Join(taskDir, "review.txt")
	doc, err := svc.CreateReview("task-123", srtPath, "Test Video", "繁體中文")
	if err != nil {
		t.Fatal(err)
	}
	err = svc.SaveReview(doc, reviewPath)
	if err != nil {
		t.Fatal(err)
	}

	finalSRT, err := svc.Approve("task-123", reviewPath)
	if err != nil {
		t.Fatalf("Approve failed: %v", err)
	}

	if finalSRT != filepath.Join(taskDir, "final.srt") {
		t.Errorf("expected final SRT at %q, got %q", filepath.Join(taskDir, "final.srt"), finalSRT)
	}

	if _, err := os.Stat(finalSRT); os.IsNotExist(err) {
		t.Error("final.srt was not created")
	}
}

func TestReviewService_Reject(t *testing.T) {
	tmpDir := t.TempDir()

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	_, err := svc.Reject("task-123", "需要重新翻譯")
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}

	status, err := svc.GetStatus("task-123")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status != hitl.StatusRejected {
		t.Errorf("expected status %q, got %q", hitl.StatusRejected, status)
	}
}

func TestReviewService_GetStatus(t *testing.T) {
	tmpDir := t.TempDir()

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	status, err := svc.GetStatus("task-123")
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status != hitl.StatusPending {
		t.Errorf("expected status %q, got %q", hitl.StatusPending, status)
	}
}
