package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/types"
	"krillin-ai/log"
)

type fakeAuditCompleter struct {
	response string
	err      error
	prompt   string
}

func (f *fakeAuditCompleter) ChatCompletion(query string) (string, error) {
	f.prompt = query
	return f.response, f.err
}

func TestAuditAndRepairTranslationsAppliesRepair(t *testing.T) {
	log.InitLogger()

	tmpDir := t.TempDir()
	completer := &fakeAuditCompleter{
		response: `{
  "items": [
    {
      "index": 0,
      "complete": false,
      "missing_meaning": ["because clause"],
      "protected_terms": ["OpenCL"],
      "should_repair": true,
      "repaired_translation": "我想看看它是否真的能重現這個 因為 OpenCL 很關鍵"
    }
  ]
}`,
	}
	svc := Service{ChatCompleter: completer}
	items := []*TranslatedItem{
		{
			OriginText:     "I want to see if it can really reproduce this because OpenCL matters",
			TranslatedText: "我想看看它是否真的能重現這個",
		},
	}

	svc.auditAndRepairTranslations(tmpDir, items, types.LanguageNameTraditionalChinese, 7)

	want := "我想看看它是否真的能重現這個 因為 OpenCL 很關鍵"
	if items[0].TranslatedText != want {
		t.Fatalf("expected repaired translation %q, got %q", want, items[0].TranslatedText)
	}
	if !strings.Contains(completer.prompt, "OpenCL") {
		t.Fatalf("expected prompt to include protected term, got %s", completer.prompt)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "translation_audit_7.json")); err != nil {
		t.Fatalf("expected audit log to be written: %v", err)
	}
}

func TestParseTranslationAuditResponseAcceptsMarkdownCodeBlock(t *testing.T) {
	response := "```json\n{\"items\":[{\"index\":1,\"complete\":true,\"should_repair\":false}]}\n```"

	got, err := parseTranslationAuditResponse(response)
	if err != nil {
		t.Fatalf("parseTranslationAuditResponse failed: %v", err)
	}
	if len(got) != 1 || got[0].Index != 1 || !got[0].Complete {
		t.Fatalf("unexpected audit response: %#v", got)
	}
}

func TestUnsafeTranslationAuditRepairRejectsCrossSentenceRepair(t *testing.T) {
	requests := []translationAuditRequestItem{{
		Index:           1,
		Source:          "not only do you need to figure out what's going on in latent space",
		Translation:     "不僅要搞清楚潛在空間裡發生了什麼",
		FollowingSource: "and deterministic space",
	}}
	result := translationAuditResult{
		Index:          1,
		MissingMeaning: []string{"deterministic space"},
	}
	if !unsafeTranslationAuditRepair(requests, result, "以及確定性空間") {
		t.Fatal("expected cross-sentence repair to be rejected")
	}
}

func TestUnsafeTranslationAuditRepairRejectsLongFillerRepair(t *testing.T) {
	requests := []translationAuditRequestItem{{
		Index:       1,
		Source:      "You know,",
		Translation: "你知道",
	}}
	result := translationAuditResult{Index: 1}
	if !unsafeTranslationAuditRepair(requests, result, "這大概就是那兩件很棒的事") {
		t.Fatal("expected long filler repair to be rejected")
	}
}

func TestExtractProtectedTermsKeepsTechnicalTerms(t *testing.T) {
	got := extractProtectedTerms("Claude uses OpenCL on Linux with xAI Grok and GPU APIs.")
	joined := strings.Join(got, ",")
	for _, want := range []string{"Claude", "OpenCL", "Linux", "xAI", "Grok", "GPU"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected protected term %q in %#v", want, got)
		}
	}
	if strings.Contains(joined, "The") {
		t.Fatalf("common capitalized words should not be protected terms: %#v", got)
	}
}
