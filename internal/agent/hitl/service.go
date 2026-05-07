package hitl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	StatusPending       = "pending"
	StatusPendingReview = "pending_review"
	StatusApproved      = "approved"
	StatusRejected      = "rejected"
)

type TaskStatus struct {
	TaskID       string     `json:"task_id"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy   string     `json:"reviewed_by,omitempty"`
	RejectReason string     `json:"reject_reason,omitempty"`
}

type ReviewService struct {
	Parser  ReviewParser
	Merger  SRTMergerInterface
	BaseDir string
}

type SRTMergerInterface interface {
	Merge(path string, segments []Segment) (MergedSRT, error)
	Write(path string, merged MergedSRT) error
}

func NewReviewService(parser ReviewParser, merger SRTMergerInterface, baseDir string) ReviewService {
	return ReviewService{
		Parser:  parser,
		Merger:  merger,
		BaseDir: baseDir,
	}
}

func (s ReviewService) CreateReview(taskID, srtPath, videoTitle, language string) (ReviewDocument, error) {
	srtFile, err := os.Open(srtPath)
	if err != nil {
		return ReviewDocument{}, err
	}
	defer srtFile.Close()

	var segments []Segment
	index := 1
	var currentStart, currentEnd time.Time
	var currentText string
	state := 0

	scanner := bufio.NewScanner(srtFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			if state == 2 && index > 0 {
				segments = append(segments, Segment{
					Index:    index,
					Start:    currentStart,
					End:      currentEnd,
					Original: currentText,
					Edited:   currentText,
				})
			}
			state = 0
			continue
		}

		if state == 0 {
			if n, err := fmt.Sscanf(line, "%d", new(int)); err == nil && n > 0 {
				index = n
				state = 1
			}
		} else if state == 1 {
			var startStr, endStr string
			if _, err := fmt.Sscanf(line, "%12s --> %12s", &startStr, &endStr); err == nil {
				currentStart, _ = parseTimestamp(startStr)
				currentEnd, _ = parseTimestamp(endStr)
				state = 2
			}
		} else if state == 2 {
			currentText = line
		}
	}

	if state == 2 && index > 0 {
		segments = append(segments, Segment{
			Index:    index,
			Start:    currentStart,
			End:      currentEnd,
			Original: currentText,
			Edited:   currentText,
		})
	}

	return ReviewDocument{
		TaskID:     taskID,
		VideoTitle: videoTitle,
		Language:   language,
		Segments:   segments,
		CreatedAt:  time.Now(),
	}, nil
}

func (s ReviewService) CreateReviewFromBilingual(taskID, bilingualSrtPath, videoTitle, language string) (ReviewDocument, error) {
	srtFile, err := os.Open(bilingualSrtPath)
	if err != nil {
		return ReviewDocument{}, err
	}
	defer srtFile.Close()

	var segments []Segment
	index := 1
	var currentStart, currentEnd time.Time
	var chineseText, englishText string
	state := 0 // 0: index, 1: time, 2: chinese, 3: english

	scanner := bufio.NewScanner(srtFile)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			if state == 3 && index > 0 {
				segments = append(segments, Segment{
					Index:    index,
					Start:    currentStart,
					End:      currentEnd,
					Original: englishText,
					Edited:   CleanPunctuation(chineseText),
				})
			}
			state = 0
			chineseText = ""
			englishText = ""
			continue
		}

		if state == 0 {
			if n, err := fmt.Sscanf(line, "%d", new(int)); err == nil && n > 0 {
				index = n
				state = 1
			}
		} else if state == 1 {
			var startStr, endStr string
			if _, err := fmt.Sscanf(line, "%12s --> %12s", &startStr, &endStr); err == nil {
				currentStart, _ = parseTimestamp(startStr)
				currentEnd, _ = parseTimestamp(endStr)
				state = 2
			}
		} else if state == 2 {
			chineseText = line
			state = 3
		} else if state == 3 {
			englishText = line
		}
	}

	if state == 3 && index > 0 {
		segments = append(segments, Segment{
			Index:    index,
			Start:    currentStart,
			End:      currentEnd,
			Original: englishText,
			Edited:   CleanPunctuation(chineseText),
		})
	}

	return ReviewDocument{
		TaskID:     taskID,
		VideoTitle: videoTitle,
		Language:   language,
		Segments:   segments,
		CreatedAt:  time.Now(),
	}, nil
}

func (s ReviewService) SaveReview(doc ReviewDocument, path string) error {
	content, err := s.Parser.Generate(doc)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func (s ReviewService) Approve(taskID, reviewPath string) (string, error) {
	doc, err := s.Parser.Parse(reviewPath)
	if err != nil {
		return "", err
	}

	taskDir := filepath.Dir(reviewPath)
	originalSRTPath := filepath.Join(taskDir, "translated.srt")

	editedSegments := make([]Segment, 0)
	for _, seg := range doc.Segments {
		cleanedEdited := CleanPunctuation(seg.Edited)
		if seg.Original != cleanedEdited || seg.Edited != seg.Original {
			editedSegments = append(editedSegments, Segment{
				Index:    seg.Index,
				Start:    seg.Start,
				End:      seg.End,
				Original: seg.Original,
				Edited:   cleanedEdited,
			})
		}
	}

	merged, err := s.Merger.Merge(originalSRTPath, editedSegments)
	if err != nil {
		return "", err
	}

	finalPath := filepath.Join(taskDir, "final.srt")
	if err := s.Merger.Write(finalPath, merged); err != nil {
		return "", err
	}

	statusPath := filepath.Join(taskDir, "status.json")
	status := TaskStatus{
		TaskID:     taskID,
		Status:     StatusApproved,
		ReviewedAt: ptr(time.Now()),
	}
	if err := writeStatus(statusPath, status); err != nil {
		return "", err
	}

	return finalPath, nil
}

func (s ReviewService) Reject(taskID, reason string) (string, error) {
	taskDir := filepath.Join(s.BaseDir, taskID)
	statusPath := filepath.Join(taskDir, "status.json")

	status := TaskStatus{
		TaskID:       taskID,
		Status:       StatusRejected,
		ReviewedAt:   ptr(time.Now()),
		RejectReason: reason,
	}

	if err := writeStatus(statusPath, status); err != nil {
		return "", err
	}

	return taskDir, nil
}

func (s ReviewService) GetStatus(taskID string) (string, error) {
	taskDir := filepath.Join(s.BaseDir, taskID)
	statusPath := filepath.Join(taskDir, "status.json")

	data, err := os.ReadFile(statusPath)
	if err != nil {
		if os.IsNotExist(err) {
			return StatusPending, nil
		}
		return "", err
	}

	var status TaskStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return "", err
	}

	return status.Status, nil
}

func (s ReviewService) SetPendingReview(taskID string) error {
	taskDir := filepath.Join(s.BaseDir, taskID)
	statusPath := filepath.Join(taskDir, "status.json")

	status := TaskStatus{
		TaskID:    taskID,
		Status:    StatusPendingReview,
		CreatedAt: time.Now(),
	}

	return writeStatus(statusPath, status)
}

func writeStatus(path string, status TaskStatus) error {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func ptr(t time.Time) *time.Time {
	return &t
}