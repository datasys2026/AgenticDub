package hitl_test

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/agent/hitl"
	"github.com/gin-gonic/gin"
)

func TestAPI_GetReview(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

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
	os.WriteFile(srtPath, []byte(srtContent), 0644)

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, err := svc.CreateReview("task-123", srtPath, "Test Video", "繁體中文")
	if err != nil {
		t.Fatal(err)
	}

	reviewPath := filepath.Join(taskDir, "review.txt")
	svc.SaveReview(doc, reviewPath)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/hitl/review/task-123", nil)
	c.Params = []gin.Param{{Key: "task_id", Value: "task-123"}}

	handler := func(c *gin.Context) {
		taskID := c.Param("task_id")
		reviewPath := filepath.Join(tmpDir, taskID, "review.txt")

		content, err := os.ReadFile(reviewPath)
		if err != nil {
			c.JSON(404, gin.H{"error": "review not found"})
			return
		}

		c.Data(200, "text/plain; charset=utf-8", content)
	}
	handler(c)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAPI_Approve(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

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
	os.WriteFile(srtPath, []byte(srtContent), 0644)

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, _ := svc.CreateReview("task-123", srtPath, "Test Video", "繁體中文")
	reviewPath := filepath.Join(taskDir, "review.txt")
	svc.SaveReview(doc, reviewPath)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/api/hitl/approve/task-123", nil)
	c.Params = []gin.Param{{Key: "task_id", Value: "task-123"}}

	handler := func(c *gin.Context) {
		taskID := c.Param("task_id")
		taskDir := filepath.Join(tmpDir, taskID)
		reviewPath := filepath.Join(taskDir, "review.txt")

		svc := hitl.ReviewService{
			Parser:  hitl.TxtParser{},
			Merger:  hitl.SRTMerger{},
			BaseDir: tmpDir,
		}

		finalPath, err := svc.Approve(taskID, reviewPath)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"final_srt": finalPath})
	}
	handler(c)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if !strings.HasSuffix(resp["final_srt"], "final.srt") {
		t.Errorf("expected final.srt in path, got %q", resp["final_srt"])
	}
}

func TestAPI_Reject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := `{"reason": "需要重新翻譯"}`
	c.Request = httptest.NewRequest("POST", "/api/hitl/reject/task-123", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = []gin.Param{{Key: "task_id", Value: "task-123"}}

	handler := func(c *gin.Context) {
		taskID := c.Param("task_id")

		var req struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}

		_, err := svc.Reject(taskID, req.Reason)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "rejected"})
	}
	handler(c)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	status, _ := svc.GetStatus("task-123")
	if status != hitl.StatusRejected {
		t.Errorf("expected status rejected, got %s", status)
	}
}

func TestAPI_GetStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tmpDir := t.TempDir()

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/hitl/status/task-123", nil)
	c.Params = []gin.Param{{Key: "task_id", Value: "task-123"}}

	handler := func(c *gin.Context) {
		taskID := c.Param("task_id")
		status, err := svc.GetStatus(taskID)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": status})
	}
	handler(c)

	if w.Code != 200 {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestReviewContent_Editable(t *testing.T) {
	tmpDir := t.TempDir()

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
	os.WriteFile(srtPath, []byte(srtContent), 0644)

	svc := hitl.ReviewService{
		Parser:  hitl.TxtParser{},
		Merger:  hitl.SRTMerger{},
		BaseDir: tmpDir,
	}

	doc, _ := svc.CreateReview("task-123", srtPath, "Test Video", "繁體中文")
	reviewPath := filepath.Join(taskDir, "review.txt")
	svc.SaveReview(doc, reviewPath)

	contentBytes, _ := os.ReadFile(reviewPath)
	content := string(contentBytes)

	t.Logf("Review content:\n%s", content)
	t.Logf("Number of segments: %d", len(doc.Segments))

	if len(doc.Segments) != 2 {
		t.Errorf("expected 2 segments, got %d", len(doc.Segments))
	}

	if len(doc.Segments) >= 1 {
		t.Logf("Segment 1: index=%d, original=%q, edited=%q", doc.Segments[0].Index, doc.Segments[0].Original, doc.Segments[0].Edited)
	}
	if len(doc.Segments) >= 2 {
		t.Logf("Segment 2: index=%d, original=%q, edited=%q", doc.Segments[1].Index, doc.Segments[1].Original, doc.Segments[1].Edited)
	}

	if len(doc.Segments) >= 2 && doc.Segments[1].Edited == "早安" {
		t.Log("Segment 2 was correctly edited")
	}
}