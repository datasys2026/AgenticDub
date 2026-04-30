package translator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ReflectiveTranslator struct {
	llm         LLMProvider
	maxRetries  int
	batchSize   int
}

func NewReflectiveTranslator(llm LLMProvider) *ReflectiveTranslator {
	return &ReflectiveTranslator{
		llm:        llm,
		maxRetries: 3,
		batchSize:  8,
	}
}

func (t *ReflectiveTranslator) DirectTranslate(ctx context.Context, chunk *Chunk) error {
	prompt := t.buildDirectPrompt(chunk)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	resp, err := t.llm.ChatCompletion(ctx, messages)
	if err != nil {
		return fmt.Errorf("direct translation failed: %w", err)
	}

	translations, err := t.parseJSONArray(resp.Content)
	if err != nil {
		return fmt.Errorf("failed to parse direct translation: %w", err)
	}

	for i, seg := range chunk.Segments {
		if i < len(translations) {
			seg.Direct = translations[i]
			chunk.Segments[i] = seg
		}
	}

	return nil
}

func (t *ReflectiveTranslator) ReflectTranslate(ctx context.Context, chunk *Chunk) error {
	prompt := t.buildReflectPrompt(chunk)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	resp, err := t.llm.ChatCompletion(ctx, messages)
	if err != nil {
		return fmt.Errorf("reflection failed: %w", err)
	}

	reflections, err := t.parseJSONArray(resp.Content)
	if err != nil {
		return fmt.Errorf("failed to parse reflection: %w", err)
	}

	for i, seg := range chunk.Segments {
		if i < len(reflections) {
			seg.Reflection = reflections[i]
			chunk.Segments[i] = seg
		}
	}

	return nil
}

func (t *ReflectiveTranslator) FinalTranslate(ctx context.Context, chunk *Chunk) error {
	prompt := t.buildFinalPrompt(chunk)

	messages := []Message{
		{Role: "user", Content: prompt},
	}

	resp, err := t.llm.ChatCompletion(ctx, messages)
	if err != nil {
		return fmt.Errorf("final translation failed: %w", err)
	}

	finalTranslations, err := t.parseJSONArray(resp.Content)
	if err != nil {
		return fmt.Errorf("failed to parse final translation: %w", err)
	}

	for i, seg := range chunk.Segments {
		if i < len(finalTranslations) {
			seg.Final = finalTranslations[i]
			chunk.Segments[i] = seg
		}
	}

	return nil
}

func (t *ReflectiveTranslator) TranslateChunk(ctx context.Context, chunk *Chunk) error {
	if err := t.DirectTranslate(ctx, chunk); err != nil {
		return err
	}
	if err := t.ReflectTranslate(ctx, chunk); err != nil {
		return err
	}
	return t.FinalTranslate(ctx, chunk)
}

func (t *ReflectiveTranslator) parseJSONArray(content string) ([]string, error) {
	content = strings.TrimSpace(content)

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSuffix(content, "```")

	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON array found")
	}

	arrayContent := content[start : end+1]

	var result []string
	if err := json.Unmarshal([]byte(arrayContent), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}

func (t *ReflectiveTranslator) buildDirectPrompt(chunk *Chunk) string {
	lines := make([]string, len(chunk.Segments))
	for i, seg := range chunk.Segments {
		lines[i] = fmt.Sprintf("%d. %s", i+1, seg.Original)
	}

	termsNote := ""
	if len(chunk.terminology) > 0 {
		termsNote = "### Terminology\n"
		for _, term := range chunk.terminology {
			termsNote += fmt.Sprintf("- %s: %s (%s)\n", term.Term, term.Translation, term.Note)
		}
	}

	return fmt.Sprintf(`You are a professional Netflix subtitle translator.

Translate the following subtitles from %s to %s.
- Translate line by line, maintaining original meaning
- Use professional terminology consistently
- Keep each translation as a single line

%s
### Subtitles to translate:
%s

Output only a JSON array of translated strings, same order.`, chunk.SourceLang, chunk.TargetLang, termsNote, strings.Join(lines, "\n"))
}

func (t *ReflectiveTranslator) buildReflectPrompt(chunk *Chunk) string {
	lines := make([]string, len(chunk.Segments))
	for i, seg := range chunk.Segments {
		lines[i] = fmt.Sprintf("%d. Original: %s\n   Direct: %s", i+1, seg.Original, seg.Direct)
	}

	return fmt.Sprintf(`You are a Netflix subtitle translator reviewing translations.

Review these direct translations from %s to %s.
For each line, provide a brief note about any issues found.
If a line is good, use an empty string "".

### Direct translations:
%s

Output a JSON array of strings, one per line. Example: ["", "too long", "unnatural phrasing"]

Output ONLY the JSON array, no other text.`, chunk.SourceLang, chunk.TargetLang, strings.Join(lines, "\n\n"))
}

func (t *ReflectiveTranslator) buildFinalPrompt(chunk *Chunk) string {
	lines := make([]string, len(chunk.Segments))
	for i, seg := range chunk.Segments {
		reflection := seg.Reflection
		if reflection == "" {
			reflection = "OK"
		}
		lines[i] = fmt.Sprintf("%d. Original: %s\n   Direct: %s\n   Issue: %s", i+1, seg.Original, seg.Direct, reflection)
	}

	return fmt.Sprintf(`Create final polished translations from %s to %s.
Rules:
- Max 42 characters per line for CJK
- Natural, conversational tone
- Apply the review notes to improve translations

### Original and Direct translations:
%s

Output only a JSON array of final translated strings.`, chunk.SourceLang, chunk.TargetLang, strings.Join(lines, "\n\n"))
}