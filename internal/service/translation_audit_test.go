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
