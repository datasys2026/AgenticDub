package stt

import (
	"context"
)

type TranscriptSegment struct {
	Start   float64   `json:"start"`
	End     float64   `json:"end"`
	Text    string    `json:"text"`
	Words   []Word    `json:"words,omitempty"`
}

type Word struct {
	Word       string  `json:"word"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Probability float64 `json:"probability,omitempty"`
}

type Transcript struct {
	Segments []TranscriptSegment `json:"segments"`
	Language string              `json:"language,omitempty"`
}

type STTProvider interface {
	Transcribe(ctx context.Context, audioPath string, language string) (*Transcript, error)
	Name() string
}