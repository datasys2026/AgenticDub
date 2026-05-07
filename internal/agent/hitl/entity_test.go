package hitl_test

import (
	"testing"
	"time"

	"krillin-ai/internal/agent/hitl"
)

func TestSegment_TimeRange(t *testing.T) {
	segment := hitl.Segment{
		Index:    1,
		Start:    time.Unix(0, 12*int64(time.Second)),
		End:      time.Unix(0, 15*int64(time.Second)),
		Original: "Hello world",
		Edited:   "你好世界",
	}

	if segment.Start.Second() != 12 {
		t.Errorf("expected Start second=12, got %d", segment.Start.Second())
	}
	if segment.End.Second() != 15 {
		t.Errorf("expected End second=15, got %d", segment.End.Second())
	}
}

func TestSegment_HasEdit(t *testing.T) {
	segmentNoEdit := hitl.Segment{
		Index:    1,
		Original: "Hello",
		Edited:   "Hello",
	}

	segmentWithEdit := hitl.Segment{
		Index:    2,
		Original: "Hello",
		Edited:   "你好",
	}

	if segmentNoEdit.HasEdit() {
		t.Error("segment with same original/edited should return false")
	}

	if !segmentWithEdit.HasEdit() {
		t.Error("segment with different original/edited should return true")
	}
}

func TestReviewDocument_Title(t *testing.T) {
	doc := hitl.ReviewDocument{
		TaskID:     "test-123",
		VideoTitle: "Test Video",
		Language:   "繁體中文",
	}

	expected := "【審核】Test Video (test-123)"
	if doc.Title() != expected {
		t.Errorf("expected title %q, got %q", expected, doc.Title())
	}
}