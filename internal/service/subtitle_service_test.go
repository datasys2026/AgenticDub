package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"krillin-ai/internal/agent/hitl"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
)

func TestWaitForReviewReturnsErrorOnReject(t *testing.T) {
	oldPollInterval := reviewPollInterval
	reviewPollInterval = 10 * time.Millisecond
	defer func() {
		reviewPollInterval = oldPollInterval
	}()

	taskID := "reject-test"
	taskDir := t.TempDir()
	taskPtr := &types.SubtitleTask{
		TaskId:     taskID,
		Status:     types.SubtitleTaskStatusPendingReview,
		ProcessPct: 90,
	}
	storage.SubtitleTasks.Store(taskID, taskPtr)
	defer storage.SubtitleTasks.Delete(taskID)

	status := hitl.TaskStatus{
		TaskID:       taskID,
		Status:       hitl.StatusRejected,
		RejectReason: "bad timestamps",
	}
	data, err := json.Marshal(status)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "status.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TaskId:       taskID,
		TaskBasePath: taskDir,
		TaskPtr:      taskPtr,
	}

	err = (Service{}).waitForReview(stepParam, filepath.Join(taskDir, "review.txt"), context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "review rejected") {
		t.Fatalf("expected review rejected error, got %v", err)
	}
	if taskPtr.Status != types.SubtitleTaskStatusFailed {
		t.Fatalf("expected failed task status, got %v", taskPtr.Status)
	}
	if taskPtr.FailReason != "bad timestamps" {
		t.Fatalf("expected reject reason to be preserved, got %q", taskPtr.FailReason)
	}
}
