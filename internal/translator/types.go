package translator

import (
	"context"
)

type Segment struct {
	Index         int
	Start         float64
	End           float64
	Original      string
	Direct        string
	Reflection    string
	Final        string
}

func (s *Segment) Duration() float64 {
	return s.End - s.Start
}

type Transcript struct {
	Segments []Segment
	Language string
}

func (t *Transcript) TotalDuration() float64 {
	if len(t.Segments) == 0 {
		return 0
	}
	return t.Segments[len(t.Segments)-1].End
}

type Chunk struct {
	Index         int
	Segments      []Segment
	SourceLang    string
	TargetLang    string
	maxChars      int
	maxSegments   int
	theme         string
	terminology   []Term
}

func NewChunk(index int, sourceLang, targetLang string) *Chunk {
	return &Chunk{
		Index:       index,
		Segments:    make([]Segment, 0),
		SourceLang:  sourceLang,
		TargetLang:  targetLang,
		maxChars:    600,
		maxSegments: 10,
	}
}

func (c *Chunk) AddSegment(seg Segment) {
	seg.Index = len(c.Segments)
	c.Segments = append(c.Segments, seg)
}

func (c *Chunk) IsFull() bool {
	if len(c.Segments) >= c.maxSegments {
		return true
	}
	if c.CharCount() >= c.maxChars {
		return true
	}
	return false
}

func (c *Chunk) CharCount() int {
	count := 0
	for _, seg := range c.Segments {
		count += len(seg.Original)
	}
	return count
}

func (c *Chunk) GetText() string {
	text := ""
	for _, seg := range c.Segments {
		text += seg.Original + "\n"
	}
	return text[:len(text)-1]
}

type Term struct {
	Term       string `json:"term"`
	Translation string `json:"translation"`
	Note       string `json:"note,omitempty"`
}

type TranslationResult struct {
	Segments []Segment
	Chunks   []*Chunk
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Content       string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type LLMProvider interface {
	ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error)
	Name() string
}

type Translator interface {
	DirectTranslate(ctx context.Context, chunk *Chunk) error
	ReflectTranslate(ctx context.Context, chunk *Chunk) error
	FinalTranslate(ctx context.Context, chunk *Chunk) error
	TranslateChunk(ctx context.Context, chunk *Chunk) error
}

type TerminologyProvider interface {
	ExtractTerms(ctx context.Context, transcript *Transcript, targetLang string) ([]Term, error)
}