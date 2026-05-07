package hitl

import (
	"fmt"
	"time"
)

type Segment struct {
	Index    int
	Start    time.Time
	End      time.Time
	Original string
	Edited   string
}

func (s Segment) HasEdit() bool {
	return s.Original != s.Edited
}

func (s Segment) TimeRange() string {
	return fmt.Sprintf("%s --> %s", formatTimestamp(s.Start), formatTimestamp(s.End))
}

type ReviewDocument struct {
	TaskID       string
	VideoTitle   string
	Language     string
	Segments     []Segment
	CreatedAt    time.Time
	ReviewedAt   *time.Time
	ReviewedBy   string
	RejectionMsg string
}

func (d ReviewDocument) Title() string {
	title := d.VideoTitle
	if title == "" {
		title = "Untitled"
	}
	return fmt.Sprintf("【審核】%s (%s)", title, d.TaskID)
}

func (d ReviewDocument) EditCount() int {
	count := 0
	for _, s := range d.Segments {
		if s.HasEdit() {
			count++
		}
	}
	return count
}

func formatTimestamp(t time.Time) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d",
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000000)
}