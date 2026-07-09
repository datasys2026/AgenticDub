package service

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"krillin-ai/config"
	"krillin-ai/internal/types"
)

type captureTTSClient struct {
	mu     sync.Mutex
	texts  []string
	voices []string
}

func (c *captureTTSClient) Text2Speech(text string, voice string, outputFile string) error {
	c.mu.Lock()
	c.texts = append(c.texts, text)
	c.voices = append(c.voices, voice)
	c.mu.Unlock()
	return os.WriteFile(outputFile, []byte("audio"), 0644)
}

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

func TestInterpreterDubTimingsDelayAndAvoidOverlap(t *testing.T) {
	subtitles := []types.SrtSentenceWithStrTime{
		{Start: "00:00:01,000", End: "00:00:02,000", Text: "第一句"},
		{Start: "00:00:02,100", End: "00:00:03,000", Text: "第二句"},
	}
	audioDurations := []float64{1.4, 0.7}

	got, err := buildInterpreterDubTimingsFromDurations(subtitles, audioDurations, 0.8, 0.12)
	if err != nil {
		t.Fatalf("buildInterpreterDubTimingsFromDurations() error = %v", err)
	}
	if math.Abs(got[0].Start-1.8) > 0.001 {
		t.Fatalf("first dub start = %.3f, want 1.800", got[0].Start)
	}
	if math.Abs(got[0].End-3.2) > 0.001 {
		t.Fatalf("first dub end = %.3f, want 3.200", got[0].End)
	}
	if math.Abs(got[1].Start-3.32) > 0.001 {
		t.Fatalf("second dub should be pushed after first plus gap, got %.3f", got[1].Start)
	}
	if math.Abs(got[1].PrecedingSilence-0.12) > 0.001 {
		t.Fatalf("second preceding silence = %.3f, want 0.120", got[1].PrecedingSilence)
	}
}

func TestWriteSRTWithDubTimingsUsesDubTimeline(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "target.srt")
	subtitles := []types.SrtSentenceWithStrTime{
		{Start: "00:00:01,000", End: "00:00:02,000", Text: "第一句"},
	}
	timings := []dubSpeechTiming{
		{Start: 1.8, End: 3.2},
	}

	if err := writeSRTWithDubTimings(path, subtitles, timings); err != nil {
		t.Fatalf("writeSRTWithDubTimings() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "00:00:01,799 --> 00:00:03,200") && !strings.Contains(content, "00:00:01,800 --> 00:00:03,200") {
		t.Fatalf("expected dub timing in SRT, got:\n%s", content)
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

func TestSpeakerVoiceAssignmentsKeepSameVoicePerSpeaker(t *testing.T) {
	subtitles := []types.SrtSentenceWithStrTime{
		{Text: "[Speaker 1]: 第一句"},
		{Text: "[Speaker 1]: 第二句"},
		{Text: "旁白內容"},
	}

	got := buildSpeakerVoiceAssignments(subtitles, "ara")
	if got["speaker-1"] != "ara" {
		t.Fatalf("expected speaker-1 to use ara, got %#v", got)
	}
	if got[defaultSpeakerID] != "ara" {
		t.Fatalf("expected narrator to use ara, got %#v", got)
	}
}

func TestBuildTTSSpeechTextKeepsSpeechTagsOutOfDisplayPath(t *testing.T) {
	display := cleanSubtitleDisplayText("[Speaker 1]: <emphasis>重要</emphasis> [pause] 說明")
	if display != "重要 說明" {
		t.Fatalf("cleanSubtitleDisplayText() = %q, want %q", display, "重要 說明")
	}

	speech := buildTTSSpeechText("這是一段關於 xAI API 與 LLM pipeline 的技術說明")
	if !strings.HasPrefix(speech, "<slow>") || !strings.HasSuffix(speech, "</slow>") {
		t.Fatalf("expected long technical speech text to be wrapped with slow tag, got %q", speech)
	}
}

func TestProcessSubtitlesUsesStableSpeakerVoiceAndSpeechText(t *testing.T) {
	tmpDir := t.TempDir()
	client := &captureTTSClient{}
	svc := Service{TtsClient: client}
	subtitles := []types.SrtSentenceWithStrTime{
		{Text: "[Speaker 1]: 第一段"},
		{Text: "[Speaker 1]: 第二段"},
	}
	assignments := buildSpeakerVoiceAssignments(subtitles, "leo")

	if err := svc.processSubtitlesConcurrently(subtitles, assignments, &types.SubtitleTaskStepParam{TaskBasePath: tmpDir}); err != nil {
		t.Fatalf("processSubtitlesConcurrently() error = %v", err)
	}
	if len(client.texts) != 2 || client.texts[0] != "第一段" || client.texts[1] != "第二段" {
		t.Fatalf("expected TTS texts without speaker prefixes, got %#v", client.texts)
	}
	if len(client.voices) != 2 || client.voices[0] != "leo" || client.voices[1] != "leo" {
		t.Fatalf("expected stable voice leo, got %#v", client.voices)
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
