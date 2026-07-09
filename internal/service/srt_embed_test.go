package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/types"
)

func TestNormalizeEmbedSubtitleVideoTypeDefaultsToOriginal(t *testing.T) {
	tests := map[string]string{
		"":         "original",
		"adaptive": "original",
		"Original": "original",
		"none":     "none",
	}

	for input, want := range tests {
		if got := normalizeEmbedSubtitleVideoType(input); got != want {
			t.Fatalf("normalizeEmbedSubtitleVideoType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveEmbedSubtitleVideoTypeKeepsOriginalAspect(t *testing.T) {
	if got := resolveEmbedSubtitleVideoType("original", 1920, 1080); got != "horizontal" {
		t.Fatalf("16:9 original should resolve to horizontal, got %q", got)
	}
	if got := resolveEmbedSubtitleVideoType("original", 720, 1280); got != "vertical" {
		t.Fatalf("9:16 original should resolve to vertical, got %q", got)
	}
	if got := resolveEmbedSubtitleVideoType("vertical", 1920, 1080); got != "vertical" {
		t.Fatalf("explicit vertical should stay vertical, got %q", got)
	}
}

func TestVerticalTransferInputUsesDubbedVideoWhenTtsEnabled(t *testing.T) {
	stepParam := &types.SubtitleTaskStepParam{
		InputVideoPath:       "origin_video.mp4",
		EnableTts:            true,
		VideoWithTtsFilePath: "video_with_tts.mp4",
	}

	input, filename := verticalTransferInput(stepParam)
	if input != "video_with_tts.mp4" {
		t.Fatalf("vertical transfer should use dubbed video input, got %q", input)
	}
	if filename != types.SubtitleTaskTransferredVerticalVideoWithTtsFileName {
		t.Fatalf("vertical transfer should use TTS-specific temp filename, got %q", filename)
	}
}

func TestVerticalTransferInputUsesOriginalVideoWithoutTts(t *testing.T) {
	stepParam := &types.SubtitleTaskStepParam{
		InputVideoPath: "origin_video.mp4",
	}

	input, filename := verticalTransferInput(stepParam)
	if input != "origin_video.mp4" {
		t.Fatalf("vertical transfer should use original video input, got %q", input)
	}
	if filename != types.SubtitleTaskTransferredVerticalVideoFileName {
		t.Fatalf("vertical transfer should use original temp filename, got %q", filename)
	}
}

func TestSplitChineseTextBalancesLinesAndAvoidsShortTail(t *testing.T) {
	got := splitChineseText("所以彼得斯坦伯格在這裡他寫了一個軟體", 10)
	if len(got) != 2 {
		t.Fatalf("expected 2 balanced lines, got %d: %#v", len(got), got)
	}
	if len([]rune(got[1])) <= 3 {
		t.Fatalf("expected no short tail line, got %#v", got)
	}
}

func TestSplitChineseTextDoesNotSplitLatinOrNumberTokens(t *testing.T) {
	got := splitChineseText("它在短短幾週就超越了 Linux 花 30 年才達到的成就", 12)
	joined := strings.Join(got, "\n")

	if strings.Contains(joined, "L\ninux") || strings.Contains(joined, "Lin\nux") || strings.Contains(joined, "Linu\nx") {
		t.Fatalf("expected Linux token to stay intact, got %#v", got)
	}
	if strings.Contains(joined, "3\n0") {
		t.Fatalf("expected number token to stay intact, got %#v", got)
	}
}

func TestSplitChineseTextDoesNotSplitSlashSeparatedTechnicalToken(t *testing.T) {
	got := splitChineseText("我都會做一個 UI/UX 的測試 看看它們有多厲害", 12)
	joined := strings.Join(got, "\n")
	if strings.Contains(joined, "UI/\nUX") || strings.Contains(joined, "UI\n/UX") {
		t.Fatalf("expected UI/UX token to stay intact, got %#v", got)
	}
}

func TestSplitChineseTextAvoidsTooShortLeadingLine(t *testing.T) {
	got := splitChineseText("Fable Five 來了 我得說", 12)
	if len(got) != 2 || got[0] != "Fable Five" {
		t.Fatalf("expected first line to avoid isolated Latin token, got %#v", got)
	}
}

func TestCleanSubtitleDisplayTextRemovesInlinePunctuation(t *testing.T) {
	got := cleanSubtitleDisplayText("所以彼得・斯坦伯格來了，他寫了一套軟體。")
	want := "所以彼得 斯坦伯格來了 他寫了一套軟體"
	if got != want {
		t.Fatalf("cleanSubtitleDisplayText() = %q, want %q", got, want)
	}
}

func TestCleanSubtitleDisplayTextRemovesSpeechTagsAndSpeakerPrefix(t *testing.T) {
	got := cleanSubtitleDisplayText("[Speaker 1]: <emphasis>這裡很重要</emphasis> [pause] 請注意")
	want := "這裡很重要 請注意"
	if got != want {
		t.Fatalf("cleanSubtitleDisplayText() = %q, want %q", got, want)
	}
}

func TestSrtToAssVerticalSplitsLongChineseBlockIntoSingleLineDialogues(t *testing.T) {
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.srt")
	output := filepath.Join(tmpDir, "output.ass")
	content := `1
00:00:00,000 --> 00:00:03,000
所以彼得斯坦伯格在這裡他寫了一個軟體

`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TargetLanguage: types.LanguageNameTraditionalChinese,
		MaxWordOneLine: 10,
	}
	if err := srtToAss(input, output, false, stepParam); err != nil {
		t.Fatalf("srtToAss failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	ass := string(data)
	if got := strings.Count(ass, "Dialogue:"); got != 2 {
		t.Fatalf("expected long subtitle block to be split into two dialogue events, got %d:\n%s", got, ass)
	}
	if strings.Contains(ass, `\N`) {
		t.Fatalf("expected no ASS line break inside dialogue events:\n%s", ass)
	}
}

func TestSrtToAssHorizontalTargetOnlyWritesDialogue(t *testing.T) {
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.srt")
	output := filepath.Join(tmpDir, "output.ass")
	content := `1
00:00:00,000 --> 00:00:03,000
這是一段單語中文字幕

`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TargetLanguage: types.LanguageNameTraditionalChinese,
		MaxWordOneLine: 12,
	}
	if err := srtToAss(input, output, true, stepParam); err != nil {
		t.Fatalf("srtToAss failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	ass := string(data)
	if got := strings.Count(ass, "Dialogue:"); got != 1 {
		t.Fatalf("expected one dialogue event for target-only horizontal subtitle, got %d:\n%s", got, ass)
	}
	if !strings.Contains(ass, `{\rMajor}`) {
		t.Fatalf("expected target-only horizontal subtitle to use Major style:\n%s", ass)
	}
}

func TestSrtToAssHorizontalSplitsLongTargetOnlySubtitleWithoutLineBreaks(t *testing.T) {
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.srt")
	output := filepath.Join(tmpDir, "output.ass")
	content := `1
00:00:00,000 --> 00:00:04,000
這是一段很長的中文字幕需要重新切成多段

`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TargetLanguage: types.LanguageNameTraditionalChinese,
		MaxWordOneLine: 10,
	}
	if err := srtToAss(input, output, true, stepParam); err != nil {
		t.Fatalf("srtToAss failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	ass := string(data)
	if got := strings.Count(ass, "Dialogue:"); got < 2 {
		t.Fatalf("expected long target-only subtitle to be split, got %d:\n%s", got, ass)
	}
	if strings.Contains(ass, `\N`) {
		t.Fatalf("expected no ASS line break for long target-only subtitle:\n%s", ass)
	}
}

func TestSrtToAssHorizontalClampsOverlappingCues(t *testing.T) {
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.srt")
	output := filepath.Join(tmpDir, "output.ass")
	content := `1
00:00:00,000 --> 00:00:03,000
第一段

2
00:00:02,000 --> 00:00:04,000
第二段

`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TargetLanguage: types.LanguageNameTraditionalChinese,
		MaxWordOneLine: 12,
	}
	if err := srtToAss(input, output, true, stepParam); err != nil {
		t.Fatalf("srtToAss failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	ass := string(data)
	if !strings.Contains(ass, "Dialogue: 0,00:00:00.00,00:00:03.00") {
		t.Fatalf("expected first cue to keep original timing:\n%s", ass)
	}
	if !strings.Contains(ass, "Dialogue: 0,00:00:03.00,00:00:04.00") {
		t.Fatalf("expected second cue to start after first cue ends:\n%s", ass)
	}
}

func TestSrtToAssVerticalTreatsChineseWithLatinTokenAsMajor(t *testing.T) {
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.srt")
	output := filepath.Join(tmpDir, "output.ass")
	content := `1
00:00:00,000 --> 00:00:03,000
我說的這個軟體叫做 OpenCL

`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TargetLanguage: types.LanguageNameTraditionalChinese,
		MaxWordOneLine: 12,
	}
	if err := srtToAss(input, output, false, stepParam); err != nil {
		t.Fatalf("srtToAss failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	ass := string(data)
	if !strings.Contains(ass, `{\rMajor}`) {
		t.Fatalf("expected Chinese subtitle with OpenCL to use Major style:\n%s", ass)
	}
	if strings.Contains(ass, `{\rMinor}`) {
		t.Fatalf("expected no Minor style for Chinese subtitle with OpenCL:\n%s", ass)
	}
	if strings.Contains(ass, "·") || strings.Contains(ass, "，") || strings.Contains(ass, "。") {
		t.Fatalf("expected display punctuation to be removed:\n%s", ass)
	}
}

func TestSrtToAssVerticalDoesNotSplitMixedTokens(t *testing.T) {
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "input.srt")
	output := filepath.Join(tmpDir, "output.ass")
	content := `1
00:00:00,000 --> 00:00:03,000
它在短短幾週就超越了 Linux 花 30 年才達到的成就

`
	if err := os.WriteFile(input, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TargetLanguage: types.LanguageNameTraditionalChinese,
		MaxWordOneLine: 12,
	}
	if err := srtToAss(input, output, false, stepParam); err != nil {
		t.Fatalf("srtToAss failed: %v", err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	ass := string(data)
	if strings.Contains(ass, `Lin\Nux`) || strings.Contains(ass, `3\N0`) {
		t.Fatalf("expected mixed Latin/number tokens to stay intact:\n%s", ass)
	}
}
