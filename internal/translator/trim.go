package translator

import (
	"context"
	"fmt"
	"strings"
)

type TextTrimmer struct {
	llm          LLMProvider
	minDuration  float64
}

func NewTextTrimmer(llm LLMProvider) *TextTrimmer {
	return &TextTrimmer{
		llm:         llm,
		minDuration: 1.0,
	}
}

func (t *TextTrimmer) Trim(text string, maxDuration float64) (string, error) {
	if maxDuration <= 0 {
		return text, nil
	}

	if len(text) <= 42 {
		return text, nil
	}

	sentences := strings.Split(text, ".")
	result := strings.Builder{}
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}
		if result.Len() > 0 {
			result.WriteString(". ")
		}
		result.WriteString(trimmed)

		if result.Len() > 42 {
			break
		}
	}

	return result.String(), nil
}

func (t *TextTrimmer) TrimWithLLM(ctx context.Context, text string, duration float64) (string, error) {
	if duration <= t.minDuration {
		return text, nil
	}

	charRate := float64(len(text)) / duration
	maxChars := int(charRate * duration * 0.9)

	if len(text) <= maxChars {
		return text, nil
	}

	prompt := fmt.Sprintf(`Trim this subtitle to fit within %d characters while keeping the meaning.

Original: %s

Output only a JSON object: {"result": "trimmed text"}`, maxChars, text)

	resp, err := t.llm.ChatCompletion(ctx, []Message{
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return text, nil
	}

	return extractTrimResult(resp.Content)
}

func extractTrimResult(content string) (string, error) {
	start := strings.Index(content, `{"result":`)
	if start == -1 {
		start = strings.Index(content, "{")
	}
	end := strings.LastIndex(content, "}")
	if end == -1 {
		end = len(content)
	}

	jsonStr := content[start : end+1]

	trimmed := strings.TrimPrefix(jsonStr, `{"result":`)
	trimmed = strings.TrimPrefix(trimmed, `"`)
	trimmed = strings.TrimSuffix(trimmed, `"}`)
	trimmed = strings.TrimSuffix(trimmed, `"}`)
	trimmed = strings.TrimSuffix(trimmed, `"`)

	return trimmed, nil
}