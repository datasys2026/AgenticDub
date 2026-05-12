package agent

import (
	"context"
)

type STTInput struct {
	AudioFile string
	Language  string
	WordDir   string
}

type STTOutput struct {
	Transcript string
}

type LLMInput struct {
	Text        string
	TargetLang  string
	Terminology []Term
}

type LLMOutput struct {
	Translation string
}

type TTSInput struct {
	Text       string
	Voice      string
	OutputFile string
}

type TTSOutput struct {
	AudioFile string
}

type STTTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input STTInput) (*STTOutput, error)
}

type LLMTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input LLMInput) (*LLMOutput, error)
}

type TTSTool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input TTSInput) (*TTSOutput, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}

type Tool interface {
	Name() string
	Description() string
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

func (r *ToolRegistry) Get(name string) Tool {
	return r.tools[name]
}

func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}
