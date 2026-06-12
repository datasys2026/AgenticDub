package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"krillin-ai/internal/types"
)

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

func TestCleanSubtitleDisplayTextRemovesInlinePunctuation(t *testing.T) {
	got := cleanSubtitleDisplayText("所以彼得・斯坦伯格來了，他寫了一套軟體。")
	want := "所以彼得 斯坦伯格來了 他寫了一套軟體"
	if got != want {
		t.Fatalf("cleanSubtitleDisplayText() = %q, want %q", got, want)
	}
}

func TestSrtToAssVerticalKeepsChineseBlockInSingleDialogue(t *testing.T) {
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
	if got := strings.Count(ass, "Dialogue:"); got != 1 {
		t.Fatalf("expected one dialogue event for one subtitle block, got %d:\n%s", got, ass)
	}
	if !strings.Contains(ass, `\N`) {
		t.Fatalf("expected ASS line break within one dialogue event:\n%s", ass)
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
