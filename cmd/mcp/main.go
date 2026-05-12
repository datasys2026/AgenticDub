package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	ServerURL string
}

var cfg = &Config{
	ServerURL: getEnv("KRILLIN_SERVER_URL", "http://127.0.0.1:8899"),
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

type TranslateVideoInput struct {
	URL            string `json:"url" jsonschema:"video URL (YouTube, Bilibili, or local file path)"`
	OriginLang     string `json:"origin_lang" jsonschema:"original language (e.g., en)"`
	TargetLang     string `json:"target_lang" jsonschema:"target language (繁體中文 or 簡體中文)"`
	Bilingual      bool   `json:"bilingual" jsonschema:"include original language subtitles"`
	TTS            bool   `json:"tts" jsonschema:"generate TTS audio"`
	Voice          string `json:"voice" jsonschema:"TTS voice name (e.g., Ryan)"`
	EmbedVideoType string `json:"embed_video_type" jsonschema:"subtitle burn type (horizontal, vertical, none)"`
}

type TranslateVideoOutput struct {
	TaskID string `json:"task_id"`
}

func TranslateVideo(ctx context.Context, req *mcp.CallToolRequest, input TranslateVideoInput) (*mcp.CallToolResult, TranslateVideoOutput, error) {
	if input.URL == "" {
		return nil, TranslateVideoOutput{}, fmt.Errorf("url is required")
	}
	if input.TargetLang == "" {
		input.TargetLang = "繁體中文"
	}
	if input.Voice == "" {
		input.Voice = "Ryan"
	}
	if input.EmbedVideoType == "" {
		input.EmbedVideoType = "none"
	}

	bilingual := 0
	if input.Bilingual {
		bilingual = 1
	}
	tts := 0
	if input.TTS {
		tts = 1
	}

	payload := map[string]any{
		"url":                        input.URL,
		"origin_lang":                input.OriginLang,
		"target_lang":                input.TargetLang,
		"bilingual":                  bilingual,
		"translation_subtitle_pos":    0,
		"modal_filter":               0,
		"tts":                        tts,
		"tts_voice_code":            input.Voice,
		"embed_subtitle_video_type": input.EmbedVideoType,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, TranslateVideoOutput{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(cfg.ServerURL+"/api/capability/subtitleTask", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, TranslateVideoOutput{}, fmt.Errorf("failed to call server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, TranslateVideoOutput{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, TranslateVideoOutput{}, fmt.Errorf("failed to decode response: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, TranslateVideoOutput{}, fmt.Errorf("invalid response format")
	}

	taskID, ok := data["task_id"].(string)
	if !ok {
		return nil, TranslateVideoOutput{}, fmt.Errorf("task_id not found in response")
	}

	return nil, TranslateVideoOutput{TaskID: taskID}, nil
}

type GetTaskStatusInput struct {
	TaskID string `json:"task_id" jsonschema:"the task ID to query"`
}

type GetTaskStatusOutput struct {
	TaskID         string `json:"task_id"`
	Status         string `json:"status"`
	ProcessPercent uint8  `json:"process_percent"`
	FailReason     string `json:"fail_reason,omitempty"`
}

func GetTaskStatus(ctx context.Context, req *mcp.CallToolRequest, input GetTaskStatusInput) (*mcp.CallToolResult, GetTaskStatusOutput, error) {
	if input.TaskID == "" {
		return nil, GetTaskStatusOutput{}, fmt.Errorf("task_id is required")
	}

	resp, err := http.Get(cfg.ServerURL + "/api/capability/subtitleTask?taskId=" + input.TaskID)
	if err != nil {
		return nil, GetTaskStatusOutput{}, fmt.Errorf("failed to call server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, GetTaskStatusOutput{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, GetTaskStatusOutput{}, fmt.Errorf("failed to decode response: %w", err)
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return nil, GetTaskStatusOutput{}, fmt.Errorf("invalid response format")
	}

	processPercent := uint8(0)
	if pct, ok := data["process_percent"].(float64); ok {
		processPercent = uint8(pct)
	}

	failReason := ""
	if reason, ok := data["fail_reason"].(string); ok {
		failReason = reason
	}

	status := "processing"
	if success, ok := data["video_info"].(map[string]any); ok && success != nil {
		status = "success"
	} else if failReason != "" {
		status = "failed"
	} else if processPercent == 90 {
		status = "pending_review"
	}

	return nil, GetTaskStatusOutput{
		TaskID:         input.TaskID,
		Status:         status,
		ProcessPercent: processPercent,
		FailReason:     failReason,
	}, nil
}

type ListTasksInput struct {
	Limit int `json:"limit" jsonschema:"maximum number of tasks to return"`
}

type ListTasksOutput struct {
	Tasks []TaskSummary `json:"tasks"`
}

type TaskSummary struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Percent uint8  `json:"percent"`
}

func ListTasks(ctx context.Context, req *mcp.CallToolRequest, input ListTasksInput) (*mcp.CallToolResult, ListTasksOutput, error) {
	if input.Limit <= 0 {
		input.Limit = 20
	}

	tasks := []TaskSummary{}

	for i := 0; i < input.Limit; i++ {
		resp, err := http.Get(cfg.ServerURL + fmt.Sprintf("/api/capability/subtitleTask?taskId=task_%d", i))
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			continue
		}
	}

	return nil, ListTasksOutput{Tasks: tasks}, nil
}

type ApproveHITLInput struct {
	TaskID string `json:"task_id" jsonschema:"the task ID to approve"`
}

type ApproveHITLOutput struct {
	Message string `json:"message"`
}

func ApproveHITL(ctx context.Context, req *mcp.CallToolRequest, input ApproveHITLInput) (*mcp.CallToolResult, ApproveHITLOutput, error) {
	if input.TaskID == "" {
		return nil, ApproveHITLOutput{}, fmt.Errorf("task_id is required")
	}

	resp, err := http.Post(cfg.ServerURL+"/api/hitl/approve/"+input.TaskID, "application/json", nil)
	if err != nil {
		return nil, ApproveHITLOutput{}, fmt.Errorf("failed to call server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, ApproveHITLOutput{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, ApproveHITLOutput{Message: "review approved, continuing TTS"}, nil
}

type RejectHITLInput struct {
	TaskID string `json:"task_id" jsonschema:"the task ID to reject"`
	Reason string `json:"reason" jsonschema:"reason for rejection"`
}

type RejectHITLOutput struct {
	Message string `json:"message"`
}

func RejectHITL(ctx context.Context, req *mcp.CallToolRequest, input RejectHITLInput) (*mcp.CallToolResult, RejectHITLOutput, error) {
	if input.TaskID == "" {
		return nil, RejectHITLOutput{}, fmt.Errorf("task_id is required")
	}

	payload := map[string]string{"reason": input.Reason}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(cfg.ServerURL+"/api/hitl/reject/"+input.TaskID, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, RejectHITLOutput{}, fmt.Errorf("failed to call server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, RejectHITLOutput{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, RejectHITLOutput{Message: "review rejected"}, nil
}

type GetReviewInput struct {
	TaskID string `json:"task_id" jsonschema:"the task ID to get review content"`
}

type GetReviewOutput struct {
	Content string `json:"content"`
}

func GetReview(ctx context.Context, req *mcp.CallToolRequest, input GetReviewInput) (*mcp.CallToolResult, GetReviewOutput, error) {
	if input.TaskID == "" {
		return nil, GetReviewOutput{}, fmt.Errorf("task_id is required")
	}

	resp, err := http.Get(cfg.ServerURL + "/api/hitl/review/" + input.TaskID)
	if err != nil {
		return nil, GetReviewOutput{}, fmt.Errorf("failed to call server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, GetReviewOutput{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, GetReviewOutput{}, fmt.Errorf("failed to read response: %w", err)
	}

	return nil, GetReviewOutput{Content: string(content)}, nil
}

type GetReviewStatusInput struct {
	TaskID string `json:"task_id" jsonschema:"the task ID to get review status"`
}

type GetReviewStatusOutput struct {
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Percent uint8  `json:"process_percent"`
}

func GetReviewStatus(ctx context.Context, req *mcp.CallToolRequest, input GetReviewStatusInput) (*mcp.CallToolResult, GetReviewStatusOutput, error) {
	if input.TaskID == "" {
		return nil, GetReviewStatusOutput{}, fmt.Errorf("task_id is required")
	}

	resp, err := http.Get(cfg.ServerURL + "/api/hitl/status/" + input.TaskID)
	if err != nil {
		return nil, GetReviewStatusOutput{}, fmt.Errorf("failed to call server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, GetReviewStatusOutput{}, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, GetReviewStatusOutput{}, fmt.Errorf("failed to decode response: %w", err)
	}

	percent := uint8(0)
	if pct, ok := result["process_percent"].(float64); ok {
		percent = uint8(pct)
	}

	status := "unknown"
	if s, ok := result["status"].(string); ok {
		status = s
	}

	return nil, GetReviewStatusOutput{
		TaskID:  input.TaskID,
		Status:  status,
		Percent: percent,
	}, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && strings.Contains(s, substr)
}

func main() {
	log.Println("Starting KrillinAI MCP Server...")
	log.Printf("Server URL: %s", cfg.ServerURL)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "krillin-ai",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "translate_video",
		Description: "Start a video translation task. Translates video to target language with optional TTS and subtitle burning.",
	}, TranslateVideo)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_task_status",
		Description: "Get the status of a translation task by task ID",
	}, GetTaskStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List all translation tasks",
	}, ListTasks)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "approve_hitl",
		Description: "Approve a pending HITL review to continue TTS synthesis",
	}, ApproveHITL)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "reject_hitl",
		Description: "Reject a pending HITL review and abort the task",
	}, RejectHITL)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_review",
		Description: "Get the review.txt content for a task in pending_review status",
	}, GetReview)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_review_status",
		Description: "Get the review status of a task",
	}, GetReviewStatus)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
