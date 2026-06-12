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
