package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/dto"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
)

func TestUploadSubtitlesStoresVideoDownloadURL(t *testing.T) {
	task := &types.SubtitleTask{TaskId: "upload-video-url"}
	stepParam := &types.SubtitleTaskStepParam{
		TaskId:                task.TaskId,
		TaskPtr:               task,
		TtsResultFilePath:     "tasks/upload-video-url/tts_final_audio.wav",
		EmbeddedVideoFilePath: "output/upload-video-url_vertical_embed.mp4",
	}

	if err := (Service{}).uploadSubtitles(context.Background(), stepParam); err != nil {
		t.Fatalf("uploadSubtitles() error = %v", err)
	}

	if got, want := task.SpeechDownloadUrl, "/api/file/tasks/upload-video-url/tts_final_audio.wav"; got != want {
		t.Fatalf("SpeechDownloadUrl = %q, want %q", got, want)
	}
	if got, want := task.VideoDownloadUrl, "/api/file/output/upload-video-url_vertical_embed.mp4"; got != want {
		t.Fatalf("VideoDownloadUrl = %q, want %q", got, want)
	}
}

func TestApplyApprovedReviewUpdatesTargetOnlySources(t *testing.T) {
	taskDir := t.TempDir()
	targetPath := filepath.Join(taskDir, types.SubtitleTaskTargetLanguageSrtFileName)
	targetContent := `1
00:00:00,000 --> 00:00:02,000
彼得·斯坦伯格，來了。

`
	if err := os.WriteFile(targetPath, []byte(targetContent), 0644); err != nil {
		t.Fatal(err)
	}

	reviewPath := filepath.Join(taskDir, "review.txt")
	reviewContent := `【第 1 句】 00:00:00,000 --> 00:00:02,000
原文：Peter Steinberger is here.
字幕：彼得·斯坦伯格，來了。

`
	if err := os.WriteFile(reviewPath, []byte(reviewContent), 0644); err != nil {
		t.Fatal(err)
	}

	task := &types.SubtitleTask{TaskId: filepath.Base(taskDir)}
	stepParam := &types.SubtitleTaskStepParam{
		TaskId:             task.TaskId,
		TaskPtr:            task,
		TaskBasePath:       taskDir,
		SubtitleResultType: types.SubtitleResultTypeTargetOnly,
	}

	if err := (Service{}).applyApprovedReview(stepParam, reviewPath); err != nil {
		t.Fatalf("applyApprovedReview() error = %v", err)
	}

	updatedTarget, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updatedTarget), "·") || strings.Contains(string(updatedTarget), "，") || strings.Contains(string(updatedTarget), "。") {
		t.Fatalf("target SRT still contains punctuation:\n%s", string(updatedTarget))
	}
	if !strings.Contains(string(updatedTarget), "彼得 斯坦伯格 來了") {
		t.Fatalf("target SRT was not updated with cleaned text:\n%s", string(updatedTarget))
	}
	if stepParam.TtsSourceFilePath != targetPath {
		t.Fatalf("TtsSourceFilePath = %q, want %q", stepParam.TtsSourceFilePath, targetPath)
	}
}

func TestGetTaskStatusReturnsVideoDownloadURL(t *testing.T) {
	task := &types.SubtitleTask{
		TaskId:           "status-video-url",
		Status:           types.SubtitleTaskStatusSuccess,
		ProcessPct:       100,
		VideoDownloadUrl: "/api/file/output/status-video-url_vertical_embed.mp4",
	}
	storage.SubtitleTasks.Store(task.TaskId, task)
	t.Cleanup(func() {
		storage.SubtitleTasks.Delete(task.TaskId)
	})

	got, err := (Service{}).GetTaskStatus(dto.GetVideoSubtitleTaskReq{TaskId: task.TaskId})
	if err != nil {
		t.Fatalf("GetTaskStatus() error = %v", err)
	}

	if got.VideoDownloadUrl != task.VideoDownloadUrl {
		t.Fatalf("VideoDownloadUrl = %q, want %q", got.VideoDownloadUrl, task.VideoDownloadUrl)
	}
}
