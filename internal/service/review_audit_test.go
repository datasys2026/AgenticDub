package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"krillin-ai/internal/agent/hitl"
	"krillin-ai/internal/types"
	"krillin-ai/log"
)

func TestAuditAndSuggestReviewAppliesRepairToReviewDocument(t *testing.T) {
	log.InitLogger()

	tmpDir := t.TempDir()
	completer := &fakeAuditCompleter{
		response: `{
  "items": [
    {
      "index": 0,
      "complete": false,
      "missing_meaning": ["winning"],
      "protected_terms": ["landing page"],
      "should_repair": true,
      "repaired_translation": "這是一個得獎級 landing page，而且轉換率更高。"
    }
  ]
}`,
	}
	svc := Service{ChatCompleter: completer}
	doc := hitl.ReviewDocument{
		TaskID:     "task-review",
		VideoTitle: "Review Video",
		Language:   "繁體中文",
		CreatedAt:  time.Now(),
		Segments: []hitl.Segment{
			{
				Index:    1,
				Start:    time.Date(0, 1, 1, 0, 0, 1, 0, time.UTC),
				End:      time.Date(0, 1, 1, 0, 0, 3, 0, time.UTC),
				Original: "This is an award winning landing page and it converts better",
				Edited:   "這是一個 landing page",
			},
		},
	}

	got := svc.auditAndSuggestReview(tmpDir, doc, types.LanguageNameTraditionalChinese)

	want := "這是一個得獎級 landing page 而且轉換率更高"
	if got.Segments[0].Edited != want {
		t.Fatalf("expected review subtitle to be repaired to %q, got %q", want, got.Segments[0].Edited)
	}
	if !strings.Contains(completer.prompt, "award winning landing page") {
		t.Fatalf("expected prompt to include source text, got %s", completer.prompt)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, reviewAuditFileName)); err != nil {
		t.Fatalf("expected review audit log to be written: %v", err)
	}
	suggested, err := os.ReadFile(filepath.Join(tmpDir, reviewSuggestedFileName))
	if err != nil {
		t.Fatalf("expected suggested review to be written: %v", err)
	}
	if !strings.Contains(string(suggested), "字幕："+want) {
		t.Fatalf("expected suggested review to contain repaired subtitle, got %s", string(suggested))
	}
}

func TestAuditAndSuggestReviewKeepsDocumentOnInvalidResponse(t *testing.T) {
	log.InitLogger()

	tmpDir := t.TempDir()
	completer := &fakeAuditCompleter{response: "not json"}
	svc := Service{ChatCompleter: completer}
	doc := hitl.ReviewDocument{
		TaskID:     "task-review",
		VideoTitle: "Review Video",
		Language:   "繁體中文",
		CreatedAt:  time.Now(),
		Segments: []hitl.Segment{
			{
				Index:    1,
				Start:    time.Date(0, 1, 1, 0, 0, 1, 0, time.UTC),
				End:      time.Date(0, 1, 1, 0, 0, 3, 0, time.UTC),
				Original: "This should stay unchanged",
				Edited:   "這要保持原樣",
			},
		},
	}

	got := svc.auditAndSuggestReview(tmpDir, doc, types.LanguageNameTraditionalChinese)

	if got.Segments[0].Edited != doc.Segments[0].Edited {
		t.Fatalf("expected invalid audit response to keep subtitle, got %q", got.Segments[0].Edited)
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, reviewAuditFileName))
	if err != nil {
		t.Fatalf("expected failed review audit log to be written: %v", err)
	}
	if !strings.Contains(string(data), "invalid character") {
		t.Fatalf("expected audit log to include parse error, got %s", string(data))
	}
}
