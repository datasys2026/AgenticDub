package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"

	"krillin-ai/config"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/util"
)

func Test_isValidSplitContent(t *testing.T) {
	t.Skip("Integration test - requires local files and LLM endpoint")

	splitContentFile := "g:\\bin\\AI\\tasks\\gdQRrtQP\\srt_no_ts_1.srt"
	originalTextFile := "g:\\bin\\AI\\tasks\\gdQRrtQP\\output\\origin_1.txt"

	splitContent, err := os.ReadFile(splitContentFile)
	if err != nil {
		t.Fatalf("读取分割内容文件失败: %v", err)
	}

	originalText, err := os.ReadFile(originalTextFile)
	if err != nil {
		t.Fatalf("读取原始文本文件失败: %v", err)
	}

	if _, err := parseAndCheckContent(string(splitContent), string(originalText)); err != nil {
		t.Errorf("parseAndCheckContent() error = %v, want nil", err)
	}
}

func loadTestConfig() bool {
	var err error
	configPath := "../../config/config.toml"
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		log.GetLogger().Info("未找到配置文件")
		return false
	} else {
		log.GetLogger().Info("已找到配置文件，从配置文件中加载配置")
		if _, err = toml.DecodeFile(configPath, &config.Conf); err != nil {
			log.GetLogger().Error("加载配置文件失败", zap.Error(err))
			return false
		}
		return true
	}
}

func initService() *Service {
	log.InitLogger()
	loadTestConfig()
	return NewService()
}

func Test_splitOriginLongSentence(t *testing.T) {
	t.Skip("Integration test - requires LLM endpoint")

	testText := "then one more thing is search for file count file explorer note count is the name of the plug in install it and once enabled you can see that now I can see how many files are in each are inside each individual folder even the nested folders are showing properly now how many files are in them"
	s := initService()
	splitTextSentences, err := s.splitOriginLongSentence(testText)
	if err != nil {
		t.Errorf("splitOriginLongSentence() error = %v, want nil", err)
	}

	fmt.Println("testText:", testText)
	for i, sentence := range splitTextSentences {
		fmt.Printf("Sentence %d: %s\n", i+1, sentence)
	}
}

func TestSplitSrtTargetOnlyUsesTargetLanguageForTTS(t *testing.T) {
	log.InitLogger()

	taskDir := t.TempDir()
	outputDir := filepath.Join(taskDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	bilingualPath := filepath.Join(taskDir, types.SubtitleTaskBilingualSrtFileName)
	bilingualContent := `1
00:00:00,000 --> 00:00:01,000
Hello world
你好世界

2
00:00:01,000 --> 00:00:02,000
Good morning
早安
`
	if err := os.WriteFile(bilingualPath, []byte(bilingualContent), 0644); err != nil {
		t.Fatal(err)
	}

	stepParam := &types.SubtitleTaskStepParam{
		TaskId:               "task-123",
		TaskBasePath:         taskDir,
		BilingualSrtFilePath: bilingualPath,
		SubtitleResultType:   types.SubtitleResultTypeTargetOnly,
		OriginLanguage:       types.LanguageNameEnglish,
		TargetLanguage:       types.LanguageNameTraditionalChinese,
		UserUILanguage:       types.LanguageNameTraditionalChinese,
	}

	if err := splitSrt(stepParam); err != nil {
		t.Fatalf("splitSrt failed: %v", err)
	}

	targetPath := filepath.Join(taskDir, types.SubtitleTaskTargetLanguageSrtFileName)
	if stepParam.TtsSourceFilePath != targetPath {
		t.Fatalf("expected TTS source target SRT %q, got %q", targetPath, stepParam.TtsSourceFilePath)
	}
	if got := subtitlePathForEmbed(stepParam); got != targetPath {
		t.Fatalf("expected embed source target SRT %q, got %q", targetPath, got)
	}

	targetContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(targetContent), "Hello world") {
		t.Fatalf("target SRT should not contain original English: %s", string(targetContent))
	}
	if !strings.Contains(string(targetContent), "你好世界") {
		t.Fatalf("target SRT should contain translated Chinese: %s", string(targetContent))
	}
}

func TestSmoothTranslatedItemsMergesShortTailFragments(t *testing.T) {
	config.Conf.App.MaxSentenceLength = 70

	items := []*TranslatedItem{
		{OriginText: "Let me talk to you about some new", TranslatedText: "讓我跟你談談一些新的"},
		{OriginText: "thing", TranslatedText: "事"},
		{OriginText: "Good morning", TranslatedText: "早安"},
	}

	got := smoothTranslatedItems(items)
	if len(got) != 2 {
		t.Fatalf("expected 2 smoothed items, got %d: %#v", len(got), got)
	}
	if got[0].OriginText != "Let me talk to you about some new thing" {
		t.Fatalf("unexpected merged origin: %q", got[0].OriginText)
	}
	if got[0].TranslatedText != "讓我跟你談談一些新的事" {
		t.Fatalf("unexpected merged translation: %q", got[0].TranslatedText)
	}
	if got[1].TranslatedText != "早安" {
		t.Fatalf("standalone short subtitle should remain separate, got %q", got[1].TranslatedText)
	}
}

func TestExtendReadableSubtitleTimingsExtendsFastShortBlock(t *testing.T) {
	blocks := []*util.SrtBlock{
		{
			Index:                  1,
			Timestamp:              "00:00:18,389 --> 00:00:18,770",
			OriginLanguageSentence: "I, it's called OpenCL, and,",
			TargetLanguageSentence: "我說的這個軟體叫做 OpenCL",
		},
		{
			Index:                  2,
			Timestamp:              "00:00:21,690 --> 00:00:22,610",
			OriginLanguageSentence: "and I don't know if he realized how successful it's gonna be",
			TargetLanguageSentence: "而且我不知道他當時有沒有意識到它會有多成功",
		},
	}

	extendReadableSubtitleTimings(blocks)

	start, end, err := parseSrtTimestampSeconds(blocks[0].Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if duration := end - start; duration < 1.49 || duration > 1.51 {
		t.Fatalf("expected first block to extend to about 1.5s, got %q", blocks[0].Timestamp)
	}
	start, end, err = parseSrtTimestampSeconds(blocks[1].Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if duration := end - start; duration < 3.49 || duration > 3.51 {
		t.Fatalf("expected second block to extend for readable CPS, got %q", blocks[1].Timestamp)
	}
}

func TestGetSentenceTimestampsRejectsInvalidEnglishMatch(t *testing.T) {
	words := []types.Word{
		{Num: 0, Text: "Nothing", Start: 10.0, End: 0},
	}

	_, _, _, err := getSentenceTimestamps(words, "Nothing.", 0, types.LanguageNameEnglish)
	if err == nil {
		t.Fatal("expected invalid timestamp match to return an error")
	}
}

func TestTimestampGeneratorFallbackUsesPositiveDuration(t *testing.T) {
	blocks := []*util.SrtBlock{
		{
			Index:                  1,
			OriginLanguageSentence: "Nothing.",
			TargetLanguageSentence: "沒有內容",
		},
		{
			Index:                  2,
			OriginLanguageSentence: "Still missing.",
			TargetLanguageSentence: "仍然缺失",
		},
	}
	words := []types.Word{
		{Num: 0, Text: "Hello", Start: 10.0, End: 10.4},
		{Num: 1, Text: "world", Start: 10.4, End: 10.8},
	}

	updated, err := NewTimestampGenerator().GenerateTimestamps(blocks, words, types.LanguageNameEnglish, 0)
	if err != nil {
		t.Fatal(err)
	}

	for _, block := range updated {
		start, end, err := parseSrtTimestampSeconds(block.Timestamp)
		if err != nil {
			t.Fatal(err)
		}
		if end <= start {
			t.Fatalf("expected positive fallback duration, got %q", block.Timestamp)
		}
	}
}

func TestTimestampGeneratorRejectsLargeForwardJump(t *testing.T) {
	blocks := []*util.SrtBlock{
		{
			Index:                  1,
			OriginLanguageSentence: "Hello",
			TargetLanguageSentence: "你好",
		},
		{
			Index:                  2,
			OriginLanguageSentence: "Nothing",
			TargetLanguageSentence: "沒事",
		},
	}
	words := []types.Word{
		{Num: 0, Text: "Hello", Start: 10.0, End: 10.4},
		{Num: 1, Text: "Nothing", Start: 100.0, End: 100.5},
	}

	updated, err := NewTimestampGenerator().GenerateTimestamps(blocks, words, types.LanguageNameEnglish, 0)
	if err != nil {
		t.Fatal(err)
	}

	start, end, err := parseSrtTimestampSeconds(updated[1].Timestamp)
	if err != nil {
		t.Fatal(err)
	}
	if start >= 100 || end >= 100 {
		t.Fatalf("expected large forward jump to use fallback timing, got %q", updated[1].Timestamp)
	}
	if end <= start {
		t.Fatalf("expected fallback timing to remain positive, got %q", updated[1].Timestamp)
	}
}

func TestNormalizeSrtFileTimingsRepairsBackwardBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "merged.srt")
	content := `1
00:05:07,609 --> 00:05:09,109
right

2
00:05:06,290 --> 00:05:07,790
This
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := normalizeSrtFileTimings(path); err != nil {
		t.Fatal(err)
	}

	normalized, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(normalized), "\n")
	_, firstEnd, err := parseSrtTimestampSeconds(lines[1])
	if err != nil {
		t.Fatal(err)
	}
	secondStart, secondEnd, err := parseSrtTimestampSeconds(lines[5])
	if err != nil {
		t.Fatal(err)
	}
	if secondStart <= firstEnd {
		t.Fatalf("expected second subtitle to move after first, got %q", lines[5])
	}
	if secondEnd <= secondStart {
		t.Fatalf("expected positive repaired duration, got %q", lines[5])
	}
}
