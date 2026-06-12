package service

import (
	"math"
	"testing"

	"krillin-ai/config"
	"krillin-ai/internal/types"
)

func TestSelectTTSVoiceKeepsExplicitVoice(t *testing.T) {
	got := selectTTSVoice("ara", []string{"eve", "sal"})
	if got != "ara" {
		t.Fatalf("expected explicit voice ara, got %q", got)
	}
}

func TestSubtitleSpeechTimingSeparatesDisplayDurationAndTrailingGap(t *testing.T) {
	subtitles := []types.SrtSentenceWithStrTime{
		{Start: "00:00:00,220", End: "00:00:01,720", Text: "讓我跟你說個新東西"},
		{Start: "00:00:09,520", End: "00:00:16,590", Text: "所以彼得斯坦伯格來了"},
	}

	start, _, duration, trailingGap, err := subtitleSpeechTiming(subtitles, 0)
	if err != nil {
		t.Fatalf("subtitleSpeechTiming() error = %v", err)
	}

	if got := start.Sub(srtZeroTime()).Seconds(); math.Abs(got-0.220) > 0.001 {
		t.Fatalf("start offset = %.3f, want 0.220", got)
	}
	if math.Abs(duration-1.500) > 0.001 {
		t.Fatalf("duration = %.3f, want 1.500", duration)
	}
	if math.Abs(trailingGap-7.800) > 0.001 {
		t.Fatalf("trailingGap = %.3f, want 7.800", trailingGap)
	}
}

func TestSelectTTSVoiceRandomCandidate(t *testing.T) {
	candidates := []string{"eve", "ara", "rex", "sal", "leo"}
	got := selectTTSVoice("", candidates)
	for _, candidate := range candidates {
		if got == candidate {
			return
		}
	}
	t.Fatalf("expected selected voice from candidates, got %q", got)
}

func TestSelectTTSVoiceFallback(t *testing.T) {
	got := selectTTSVoice("", nil)
	if got != "Ryan" {
		t.Fatalf("expected Ryan fallback, got %q", got)
	}
}

func TestCompactVoiceCandidatesTrimsAndDeduplicates(t *testing.T) {
	got := compactVoiceCandidates([]string{" eve ", "", "ara", "eve"})
	if len(got) != 2 || got[0] != "eve" || got[1] != "ara" {
		t.Fatalf("unexpected compacted candidates: %#v", got)
	}
}

func TestTTSVoiceCandidatesUsesConfiguredVoices(t *testing.T) {
	conf := config.Conf
	conf.Tts.Provider = "xai-oauth"
	conf.Tts.Voices = []string{"ara", "sal"}

	got := ttsVoiceCandidates(conf)
	if len(got) != 2 || got[0] != "ara" || got[1] != "sal" {
		t.Fatalf("expected configured voices, got %#v", got)
	}
}

func TestTTSVoiceCandidatesFallsBackForXAI(t *testing.T) {
	conf := config.Conf
	conf.Tts.Provider = "xai-oauth"
	conf.Tts.Voices = nil

	got := ttsVoiceCandidates(conf)
	if len(got) != 5 || got[0] != "eve" || got[4] != "leo" {
		t.Fatalf("expected built-in xAI voices, got %#v", got)
	}
}
