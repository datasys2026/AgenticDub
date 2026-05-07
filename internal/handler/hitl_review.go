package handler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"krillin-ai/internal/agent/hitl"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"

	"github.com/gin-gonic/gin"
)

func (h Handler) GetReview(c *gin.Context) {
	taskID := c.Param("task_id")

	// Get task from storage
	task, ok := storage.SubtitleTasks.Load(taskID)
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	taskPtr := task.(*types.SubtitleTask)

	// Check if task is in pending review status
	if taskPtr.Status != types.SubtitleTaskStatusPendingReview {
		c.JSON(400, gin.H{"error": "task is not in pending review status", "status": taskPtr.Status})
		return
	}

	// Read review.txt
	reviewPath := filepath.Join("tasks", taskID, "review.txt")
	content, err := os.ReadFile(reviewPath)
	if err != nil {
		c.JSON(404, gin.H{"error": "review file not found"})
		return
	}

	c.Data(200, "text/plain; charset=utf-8", content)
}

func (h Handler) ApproveReview(c *gin.Context) {
	taskID := c.Param("task_id")

	// Get task from storage
	task, ok := storage.SubtitleTasks.Load(taskID)
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	taskPtr := task.(*types.SubtitleTask)

	if taskPtr.Status != types.SubtitleTaskStatusPendingReview {
		c.JSON(400, gin.H{"error": "task is not in pending review status"})
		return
	}

	// Write approval status
	statusPath := filepath.Join("tasks", taskID, "status.json")
	status := hitl.TaskStatus{
		TaskID:   taskID,
		Status:   hitl.StatusApproved,
		ReviewedAt: ptr(time.Now()),
	}
	data, _ := json.MarshalIndent(status, "", "  ")
	os.WriteFile(statusPath, data, 0644)

	// Update task status to continue processing
	taskPtr.Status = types.SubtitleTaskStatusProcessing
	taskPtr.ProcessPct = 91

	c.JSON(200, gin.H{"message": "review approved, continuing TTS"})
}

func (h Handler) RejectReview(c *gin.Context) {
	taskID := c.Param("task_id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	// Get task from storage
	task, ok := storage.SubtitleTasks.Load(taskID)
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	taskPtr := task.(*types.SubtitleTask)

	// Write rejection status
	statusPath := filepath.Join("tasks", taskID, "status.json")
	status := hitl.TaskStatus{
		TaskID:       taskID,
		Status:       hitl.StatusRejected,
		ReviewedAt:   ptr(time.Now()),
		RejectReason: req.Reason,
	}
	data, _ := json.MarshalIndent(status, "", "  ")
	os.WriteFile(statusPath, data, 0644)

	// Update task status
	taskPtr.Status = types.SubtitleTaskStatusFailed
	taskPtr.FailReason = req.Reason

	c.JSON(200, gin.H{"message": "review rejected"})
}

func (h Handler) GetReviewStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	// Get task from storage
	task, ok := storage.SubtitleTasks.Load(taskID)
	if !ok {
		c.JSON(404, gin.H{"error": "task not found"})
		return
	}
	taskPtr := task.(*types.SubtitleTask)

	c.JSON(200, gin.H{
		"task_id":          taskID,
		"status":           taskPtr.Status,
		"process_percent":  taskPtr.ProcessPct,
	})
}

func ptr(t time.Time) *time.Time {
	return &t
}