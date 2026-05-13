package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"krillin-ai/internal/providers/llm"
)

type MockPlannerLLM struct{}

func (m *MockPlannerLLM) ChatCompletion(ctx context.Context, messages []llm.Message) (*llm.ChatCompletionResponse, error) {
	content := messages[0].Content

	if strings.Contains(content, "分析以下影片") {
		return &llm.ChatCompletionResponse{
			Content: `{"language":"en","domain":"tech","speaker_count":1,"has_music":false,"complexity":"medium"}`,
		}, nil
	}

	if strings.Contains(content, "規劃翻譯流程") {
		return &llm.ChatCompletionResponse{
			Content: `{"strategy":"reflective","terminology_extraction":true,"voice_cloning":false,"tts_voice":"default","batch_size":8,"max_chars_cjk":42,"max_chars_latin":47,"concurrent_tts":4,"priority":"quality","steps":["stt","translate","tts","compose"]}`,
		}, nil
	}

	if strings.Contains(content, "提取需要一致翻譯的術語") {
		return &llm.ChatCompletionResponse{
			Content: `[{"term":"AI","translation":"人工智慧","note":"技術術語"},{"term":"GPU","translation":"顯示卡","note":"硬體術語"}]`,
		}, nil
	}

	return &llm.ChatCompletionResponse{
		Content: "mock response",
	}, nil
}

func (m *MockPlannerLLM) Name() string {
	return "mock-planner-llm"
}

func TestPlanner_AnalyzeVideo(t *testing.T) {
	planner := NewPlanner(&MockPlannerLLM{})
	ctx := context.Background()

	analysis, err := planner.AnalyzeVideo(ctx, "This is a test video about AI technology...")
	if err != nil {
		t.Fatalf("AnalyzeVideo failed: %v", err)
	}

	if analysis.Language != "en" {
		t.Errorf("expected language 'en', got '%s'", analysis.Language)
	}
	if analysis.Domain != "tech" {
		t.Errorf("expected domain 'tech', got '%s'", analysis.Domain)
	}
	if analysis.SpeakerCount != 1 {
		t.Errorf("expected speaker_count 1, got %d", analysis.SpeakerCount)
	}
}

func TestPlanner_CreatePlan(t *testing.T) {
	planner := NewPlanner(&MockPlannerLLM{})
	ctx := context.Background()

	analysis := &VideoAnalysis{
		Language:     "en",
		Domain:       "tech",
		SpeakerCount: 1,
		HasMusic:     false,
		Complexity:   "medium",
	}

	plan, err := planner.CreatePlan(ctx, analysis)
	if err != nil {
		t.Fatalf("CreatePlan failed: %v", err)
	}

	if plan.Strategy != StrategyReflective {
		t.Errorf("expected strategy 'reflective', got '%s'", plan.Strategy)
	}
	if plan.BatchSize != 8 {
		t.Errorf("expected batch_size 8, got %d", plan.BatchSize)
	}
	if plan.MaxCharsCJK != 42 {
		t.Errorf("expected max_chars_cjk 42, got %d", plan.MaxCharsCJK)
	}
}

func TestPlanner_ExtractTerms(t *testing.T) {
	planner := NewPlanner(&MockPlannerLLM{})
	ctx := context.Background()

	terms, err := planner.ExtractTerms(ctx, "AI and GPU are important", "繁體中文")
	if err != nil {
		t.Fatalf("ExtractTerms failed: %v", err)
	}

	if len(terms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(terms))
	}
	if terms[0].Term != "AI" {
		t.Errorf("expected first term 'AI', got '%s'", terms[0].Term)
	}
}

func TestExtractJSON(t *testing.T) {
	t.Run("extract JSON from markdown", func(t *testing.T) {
		content := "```json\n{\"key\": \"value\"}\n```"
		result := extractJSON(content)
		if result != `{"key": "value"}` {
			t.Errorf("expected JSON object, got '%s'", result)
		}
	})

	t.Run("extract JSON with braces", func(t *testing.T) {
		content := "Here is the result: {\"key\": \"value\"} end"
		result := extractJSON(content)
		if !strings.Contains(result, `"key"`) {
			t.Errorf("expected JSON object, got '%s'", result)
		}
	})

	t.Run("extract JSON array", func(t *testing.T) {
		content := "```json\n[1, 2, 3]\n```"
		result := extractJSON(content)
		if result != `[1, 2, 3]` {
			t.Errorf("expected JSON array, got '%s'", result)
		}
	})
}

func TestTranslationPlan_Defaults(t *testing.T) {
	plan := &TranslationPlan{
		Strategy:              StrategyFast,
		TerminologyExtraction: false,
		BatchSize:             8,
		MaxCharsCJK:           42,
		MaxCharsLatin:         47,
		ConcurrentTTS:          4,
	}

	if plan.Strategy != StrategyFast {
		t.Error("expected fast strategy")
	}
	if plan.BatchSize != 8 {
		t.Error("expected batch size 8")
	}
}

func TestVideoAnalysis_Fields(t *testing.T) {
	analysis := &VideoAnalysis{
		Language:     "en",
		Domain:       "education",
		Duration:     600.0,
		HasMusic:     true,
		SpeakerCount: 2,
		Complexity:   "complex",
	}

	if analysis.Language != "en" {
		t.Errorf("expected 'en', got '%s'", analysis.Language)
	}
	if analysis.SpeakerCount != 2 {
		t.Errorf("expected 2, got %d", analysis.SpeakerCount)
	}
	if !analysis.HasMusic {
		t.Error("expected HasMusic to be true")
	}
}

type MockPlannerLLMErrors struct{}

func (m *MockPlannerLLMErrors) ChatCompletion(ctx context.Context, messages []llm.Message) (*llm.ChatCompletionResponse, error) {
	return nil, errors.New("LLM error")
}

func (m *MockPlannerLLMErrors) Name() string {
	return "mock-error-llm"
}

func TestPlanner_LLMError(t *testing.T) {
	planner := NewPlanner(&MockPlannerLLMErrors{})
	ctx := context.Background()

	_, err := planner.AnalyzeVideo(ctx, "test")
	if err == nil {
		t.Error("expected error from LLM")
	}
}

func TestPlanner_ExtractTermsReturnsValidTerms(t *testing.T) {
	planner := NewPlanner(&MockPlannerLLM{})
	ctx := context.Background()

	terms, err := planner.ExtractTerms(ctx, "AI and GPU technology", "繁體中文")
	if err != nil {
		t.Fatalf("ExtractTerms failed unexpectedly: %v", err)
	}
	if len(terms) != 2 {
		t.Errorf("expected 2 terms, got %d", len(terms))
	}
}