package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"krillin-ai/internal/providers/llm"
)

type TranslationStrategy string

const (
	StrategyFast       TranslationStrategy = "fast"
	StrategyReflective TranslationStrategy = "reflective"
)

type VideoAnalysis struct {
	Language       string  `json:"language"`
	Domain         string  `json:"domain"`
	Duration       float64 `json:"duration,omitempty"`
	HasMusic       bool    `json:"has_music"`
	SpeakerCount   int     `json:"speaker_count"`
	Complexity     string  `json:"complexity"`
	TargetLanguage string  `json:"target_language,omitempty"`
}

type TranslationPlan struct {
	Strategy              TranslationStrategy `json:"strategy"`
	TerminologyExtraction bool                `json:"terminology_extraction"`
	VoiceCloning          bool                `json:"voice_cloning"`
	TTSVoice              string              `json:"tts_voice"`
	BatchSize             int                 `json:"batch_size"`
	MaxCharsCJK           int                 `json:"max_chars_cjk"`
	MaxCharsLatin         int                 `json:"max_chars_latin"`
	ConcurrentTTS          int                 `json:"concurrent_tts"`
	Priority              string              `json:"priority"`
	Steps                 []string            `json:"steps"`
}

type Planner struct {
	llm llm.LLMProvider
}

func NewPlanner(llm llm.LLMProvider) *Planner {
	return &Planner{llm: llm}
}

func (p *Planner) AnalyzeVideo(ctx context.Context, transcript string) (*VideoAnalysis, error) {
	prompt := fmt.Sprintf(`分析以下影片文字內容，回答以下問題（JSON格式）：
1. 原始語言（language）
2. 領域/類型（domain）：tech/education/entertainment/news/business等
3. 講者數量（speaker_count）：1或2以上
4. 是否有背景音樂（has_music）：true/false
5. 複雜度（complexity）：simple/medium/complex

文字內容：
%s

只輸出 JSON，不要其他文字。`, transcript)

	resp, err := p.llm.ChatCompletion(ctx, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(resp.Content)
	var analysis VideoAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, ErrInvalidAnalysis
	}

	return &analysis, nil
}

func (p *Planner) CreatePlan(ctx context.Context, analysis *VideoAnalysis) (*TranslationPlan, error) {
	prompt := fmt.Sprintf(`根據以下影片分析，規劃翻譯流程：

語言：%s
領域：%s
講者數：%d
有背景音樂：%t
複雜度：%s
目標語言：繁體中文

請輸出JSON格式的翻譯計劃，包含：
- strategy: "fast" 或 "reflective"
- terminology_extraction: true/false
- voice_cloning: true/false
- tts_voice: 建議的TTS音色
- batch_size: 每批處理的字幕數
- max_chars_cjk: 每行最大中文字數
- max_chars_latin: 每行最大英文字數
- concurrent_tts: 並行TTS數量
- priority: "quality" 或 "speed"
- steps: 執行步驟陣列

只輸出JSON。`, analysis.Language, analysis.Domain, analysis.SpeakerCount, analysis.HasMusic, analysis.Complexity)

	resp, err := p.llm.ChatCompletion(ctx, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(resp.Content)
	var plan TranslationPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, ErrInvalidPlan
	}

	return &plan, nil
}

func (p *Planner) ExtractTerms(ctx context.Context, transcript, targetLang string) ([]llm.Term, error) {
	prompt := fmt.Sprintf(`從以下文字內容中提取需要一致翻譯的術語，輸出JSON陣列：
	每個術語包含：term(原文), translation(譯文), note(備註)

	文字：
	%s

	目標語言：%s

	只輸出JSON陣列。`, transcript, targetLang)

	resp, err := p.llm.ChatCompletion(ctx, []llm.Message{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSON(resp.Content)
	var terms []llm.Term
	if err := json.Unmarshal([]byte(jsonStr), &terms); err != nil {
		return nil, ErrInvalidTerms
	}

	return terms, nil
}

func extractJSON(content string) string {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "```json") {
		content = content[7:]
	}
	if strings.HasPrefix(content, "```") {
		content = content[3:]
	}
	if strings.HasSuffix(content, "```") {
		content = content[:len(content)-3]
	}

	if strings.HasPrefix(content, "[") {
		end := strings.LastIndex(content, "]")
		if end != -1 {
			return content[:end+1]
		}
	}

	if strings.HasPrefix(content, "{") {
		end := strings.LastIndex(content, "}")
		if end != -1 {
			return content[:end+1]
		}
	}

	start := strings.Index(content, "{")
	if start == -1 {
		start = strings.Index(content, "[")
	}
	end := strings.LastIndex(content, "}")
	if end == -1 {
		end = strings.LastIndex(content, "]")
	}

	if start != -1 && end != -1 {
		return content[start : end+1]
	}

	return content
}

var (
	ErrInvalidAnalysis = errors.New("invalid video analysis")
	ErrInvalidPlan     = errors.New("invalid translation plan")
	ErrInvalidTerms    = errors.New("invalid terminology extraction")
)