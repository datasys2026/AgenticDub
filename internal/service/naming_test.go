package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"krillin-ai/internal/dto"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
)

func TestBuildTaskIDUsesReadableYouTubeParts(t *testing.T) {
	req := dto.StartVideoSubtitleTaskReq{
		Url:        "https://www.youtube.com/watch?v=tSg3FAdWvzI",
		TargetLang: "繁體中文",
	}

	got := buildTaskID(req, time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC))
	want := "youtube_tSg3FAdWvzI_zh_tw_2026-06-12"
	if got != want {
		t.Fatalf("buildTaskID() = %q, want %q", got, want)
	}
}

func TestBuildTaskIDUsesLocalFileName(t *testing.T) {
	req := dto.StartVideoSubtitleTaskReq{
		Url:        "local:/tmp/Claude Fable UI UX Review.mp4",
		TargetLang: "zh_tw",
	}

	got := buildTaskID(req, time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC))
	want := "local_claude-fable-ui-ux-review_zh_tw_2026-06-12"
	if got != want {
		t.Fatalf("buildTaskID() = %q, want %q", got, want)
	}
}

func TestUniqueTaskIDAppendsDeterministicSuffixOnCollision(t *testing.T) {
	taskID := "youtube_tSg3FAdWvzI_zh_tw_2026-06-12"
	storage.SubtitleTasks.Store(taskID, &types.SubtitleTask{TaskId: taskID})
	defer storage.SubtitleTasks.Delete(taskID)

	got := uniqueTaskID(taskID)
	want := taskID + "-2"
	if got != want {
		t.Fatalf("uniqueTaskID() = %q, want %q", got, want)
	}
}

func TestUniqueTaskIDChecksExistingTaskDirectory(t *testing.T) {
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Fatal(err)
		}
	}()

	taskID := "local-demo-zh-tw-2026-06-12"
	if err := os.MkdirAll(filepath.Join("tasks", taskID), 0755); err != nil {
		t.Fatal(err)
	}

	got := uniqueTaskID(taskID)
	want := taskID + "-2"
	if got != want {
		t.Fatalf("uniqueTaskID() = %q, want %q", got, want)
	}
}

func TestBuildEmbeddedVideoFileNameUsesReadableParts(t *testing.T) {
	stepParam := &types.SubtitleTaskStepParam{
		Link:           "https://www.youtube.com/watch?v=tSg3FAdWvzI",
		TargetLanguage: types.LanguageNameTraditionalChinese,
		EnableTts:      true,
		TaskPtr: &types.SubtitleTask{
			Title: "Claude Fable 5 UI/UX One-Shots",
		},
	}

	got := buildEmbeddedVideoFileName(stepParam, true, time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC))
	want := "claude-fable-5-ui-ux-one-shots_youtube-tSg3FAdWvzI_zh_tw_horizontal_dubbed_2026-06-12.mp4"
	if got != want {
		t.Fatalf("buildEmbeddedVideoFileName() = %q, want %q", got, want)
	}
	if strings.Contains(got, "?") || strings.Contains(got, " ") {
		t.Fatalf("filename should be shell and URL friendly, got %q", got)
	}
}

func TestBuildEmbeddedVideoFileNameFallsBackToSourceWhenTitleMissing(t *testing.T) {
	stepParam := &types.SubtitleTaskStepParam{
		Link:           "https://www.youtube.com/watch?v=tSg3FAdWvzI",
		TargetLanguage: types.LanguageNameTraditionalChinese,
		EnableTts:      true,
		TaskPtr:        &types.SubtitleTask{},
	}

	got := buildEmbeddedVideoFileName(stepParam, true, time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC))
	want := "youtube_tSg3FAdWvzI_zh_tw_horizontal_dubbed_2026-06-12.mp4"
	if got != want {
		t.Fatalf("buildEmbeddedVideoFileName() = %q, want %q", got, want)
	}
}
