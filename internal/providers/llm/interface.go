package llm

import (
	"context"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	Content string `json:"content"`
	Reasoning string `json:"reasoning,omitempty"`
}

type LLMProvider interface {
	ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error)
	Name() string
}

type TranslationRequest struct {
	SourceLang   string
	TargetLang   string
	Segments     []string
	Strategy     string // "fast" or "reflective"
	Terminology  []Term
	MaxCharsCJK  int
	MaxCharsLatin int
}

type Term struct {
	Term       string `json:"term"`
	Translation string `json:"translation"`
	Note       string `json:"note,omitempty"`
}

type TranslatedSegment struct {
	Index            int
	Original         string
	Direct           string `json:"direct,omitempty"`
	Reflection       string `json:"reflection,omitempty"`
	Final            string `json:"final"`
	StartTime        float64
	EndTime          float64
}

type TranslationResponse struct {
	Segments []TranslatedSegment
}