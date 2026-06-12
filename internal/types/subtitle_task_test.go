package types

import (
	"strings"
	"testing"
)

func TestSplitOriginLongSentencePromptRejectsTinyFragments(t *testing.T) {
	if !strings.Contains(SplitOriginLongSentencePrompt, "single-word or two-word fragments") {
		t.Fatalf("expected prompt to reject tiny fragments")
	}
	if !strings.Contains(SplitOriginLongSentencePrompt, "at least 3-5 words") {
		t.Fatalf("expected prompt to require minimum split length")
	}
}
