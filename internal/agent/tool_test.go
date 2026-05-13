package agent

import (
	"context"
	"errors"
	"testing"

	llmprovider "krillin-ai/internal/providers/llm"
)

type MockSTTTool struct {
	result string
	err    error
}

func (m *MockSTTTool) Name() string              { return "mock-stt" }
func (m *MockSTTTool) Description() string        { return "Speech to text tool" }
func (m *MockSTTTool) Execute(ctx context.Context, input STTInput) (*STTOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &STTOutput{Transcript: m.result}, nil
}

type MockLLMTool struct {
	result string
	err    error
}

func (m *MockLLMTool) Name() string               { return "mock-llm" }
func (m *MockLLMTool) Description() string         { return "Language model translation tool" }
func (m *MockLLMTool) Execute(ctx context.Context, input LLMInput) (*LLMOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &LLMOutput{Translation: m.result}, nil
}

type MockTTSTool struct {
	result string
	err    error
}

func (m *MockTTSTool) Name() string               { return "mock-tts" }
func (m *MockTTSTool) Description() string        { return "Text to speech tool" }
func (m *MockTTSTool) Execute(ctx context.Context, input TTSInput) (*TTSOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &TTSOutput{AudioFile: m.result}, nil
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	stt := &MockSTTTool{result: "transcript"}
	llm := &MockLLMTool{result: "translation"}
	tts := &MockTTSTool{result: "audio.wav"}

	registry.Register(stt)
	registry.Register(llm)
	registry.Register(tts)

	if len(registry.tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(registry.tools))
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	stt := &MockSTTTool{result: "transcript"}
	llm := &MockLLMTool{result: "translation"}

	registry.Register(stt)
	registry.Register(llm)

	found := registry.Get("mock-stt")
	if found == nil {
		t.Error("expected to find mock-stt tool")
	}

	notFound := registry.Get("non-existent")
	if notFound != nil {
		t.Error("expected nil for non-existent tool")
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	stt := &MockSTTTool{result: "transcript"}
	llm := &MockLLMTool{result: "translation"}

	registry.Register(stt)
	registry.Register(llm)

	tools := registry.List()
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestSTTTool_Execute(t *testing.T) {
	stt := &MockSTTTool{result: "Hello world"}

	output, err := stt.Execute(context.Background(), STTInput{
		AudioFile: "test.wav",
		Language:  "en",
	})

	if err != nil {
		t.Fatalf("STT Execute failed: %v", err)
	}
	if output.Transcript != "Hello world" {
		t.Errorf("expected 'Hello world', got '%s'", output.Transcript)
	}
}

func TestSTTTool_ExecuteError(t *testing.T) {
	stt := &MockSTTTool{err: errors.New("STT failed")}

	_, err := stt.Execute(context.Background(), STTInput{
		AudioFile: "test.wav",
		Language:  "en",
	})

	if err == nil {
		t.Error("expected error from STT")
	}
}

func TestLLMTool_Execute(t *testing.T) {
	llm := &MockLLMTool{result: "你好世界"}

	output, err := llm.Execute(context.Background(), LLMInput{
		Text:        "Hello world",
		TargetLang:  "繁體中文",
		Terminology: []llmprovider.Term{},
	})

	if err != nil {
		t.Fatalf("LLM Execute failed: %v", err)
	}
	if output.Translation != "你好世界" {
		t.Errorf("expected '你好世界', got '%s'", output.Translation)
	}
}

func TestTTSTool_Execute(t *testing.T) {
	tts := &MockTTSTool{result: "output.wav"}

	output, err := tts.Execute(context.Background(), TTSInput{
		Text:      "你好世界",
		Voice:     "Ryan",
		OutputFile: "output.wav",
	})

	if err != nil {
		t.Fatalf("TTS Execute failed: %v", err)
	}
	if output.AudioFile != "output.wav" {
		t.Errorf("expected 'output.wav', got '%s'", output.AudioFile)
	}
}

func TestToolMetadata(t *testing.T) {
	stt := &MockSTTTool{result: "transcript"}

	if stt.Name() != "mock-stt" {
		t.Errorf("expected name 'mock-stt', got '%s'", stt.Name())
	}
	if stt.Description() != "Speech to text tool" {
		t.Errorf("expected description 'Speech to text tool', got '%s'", stt.Description())
	}
}
