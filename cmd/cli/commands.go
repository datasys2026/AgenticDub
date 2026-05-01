package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"krillin-ai/internal/translator"
)

var (
	targetLang  string
	outputDir   string
	model       string
	strategy    string
	verbose     bool
	apiKey      string
	voice       string
	language    string
	ttsModel    string
)

const (
	STTEndpoint = "http://localhost:8006/v1/audio/transcriptions"
	LLMEndpoint = "http://localhost:4000/v1/chat/completions"
	TTSEndpoint = "http://localhost:8002/v1/audio/speech"
)

var rootCmd = &cobra.Command{
	Use:   "krilin-ai",
	Short: "AI 影片翻譯配音工具",
	Long:  `KrillinAI - 影片翻譯配音工具`,
}

var runCmd = &cobra.Command{
	Use:   "run <input>",
	Short: "執行完整翻譯流程",
	Long:  `執行影片翻譯配音流程。`,
	Args:  cobra.ExactArgs(1),
	Run:  runVideo,
}

func runVideo(cmd *cobra.Command, args []string) {
	input := args[0]

	if apiKey == "" {
		apiKey = os.Getenv("LITELLM_API_KEY")
		if apiKey == "" {
			apiKey = "datasys2026"
		}
	}

	fmt.Printf("🎬 開始處理影片: %s\n", input)
	fmt.Printf("   目標語言: %s\n", targetLang)
	fmt.Printf("   輸出目錄: %s\n", outputDir)
	fmt.Printf("   翻譯策略: %s\n", strategy)
	fmt.Printf("   TTS 模型: %s (%s - %s)\n", ttsModel, voice, language)

	os.MkdirAll(outputDir, 0755)

	audioFile, err := extractAudio(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 音頻提取失敗: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(audioFile)
	fmt.Printf("✅ 音頻已提取: %s\n", audioFile)

	fmt.Printf("🔄 正在進行語音辨識 (STT)...\n")
	segments, err := transcribeToSegments(audioFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ STT 失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ STT 完成 (%d 個片段)\n", len(segments))

	transcript := &translator.Transcript{
		Segments: segments,
		Language: "en",
	}

	fmt.Printf("🔄 正在翻譯字幕 (3-step)...\n")
	translatedSegments, err := translateAll(transcript)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 翻譯失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 翻譯完成\n")

	srtFile := filepath.Join(outputDir, "subtitles.srt")
	gen := translator.NewSRTGenerator()
	srtContent := gen.Generate(translatedSegments)
	os.WriteFile(srtFile, []byte(srtContent), 0644)
	fmt.Printf("✅ 字幕已生成: %s\n", srtFile)

	fmt.Printf("🔄 正在生成語音 (TTS)...\n")
	dubbedFile, err := synthesizeAll(translatedSegments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ TTS 失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 語音已生成: %s\n", dubbedFile)

	fmt.Printf("🔄 正在燒錄字幕到影片...\n")
	mergedFile, err := burnSubtitles(input, dubbedFile, srtFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 燒錄字幕失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 字幕已燒錄: %s\n", mergedFile)

	fmt.Printf("\n✅ 流程完成!\n")
	fmt.Printf("   字幕檔: %s\n", srtFile)
	fmt.Printf("   配音檔: %s\n", dubbedFile)
	fmt.Printf("   影片檔: %s\n", mergedFile)
}

type STTResult struct {
	Text        string  `json:"text"`
	Language    string  `json:"language"`
	Duration    float64 `json:"duration"`
	Segments    []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

func extractAudio(input string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get CWD: %w", err)
	}
	fmt.Printf("   📂 CWD: %s\n", cwd)
	fmt.Printf("   📂 Input arg: %s\n", input)

	absInput, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}
	fmt.Printf("   📂 Input: %s -> %s\n", input, absInput)
	fmt.Printf("   📂 Output dir: %s -> %s\n", outputDir, absOutputDir)

	os.MkdirAll(absOutputDir, 0755)
	audioFile := filepath.Join(absOutputDir, "audio.wav")
	fmt.Printf("   📂 Audio output: %s\n", audioFile)

	var cmd *exec.Cmd
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		fmt.Printf("   📥 下載影片...\n")
		videoFile := filepath.Join(absOutputDir, "video.mp4")
		cmd = exec.Command("yt-dlp", "-f", "best[ext=mp4]/best",
			"-o", videoFile, input)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("yt-dlp failed: %w", err)
		}
		defer os.Remove(videoFile)

		cmd = exec.Command("ffmpeg", "-i", videoFile,
			"-vn", "-acodec", "pcm_s16le", "-ac", "1", "-ar", "16000",
			audioFile, "-y")
	} else {
		cmd = exec.Command("ffmpeg", "-i", absInput,
			"-vn", "-acodec", "pcm_s16le", "-ac", "1", "-ar", "16000",
			audioFile, "-y")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("   🔧 Running FFmpeg...\n")
	runErr := cmd.Run()
	if runErr != nil {
		fmt.Printf("   ❌ FFmpeg error: %v\n", runErr)
		return "", fmt.Errorf("ffmpeg failed: %w", runErr)
	}

	fmt.Printf("   ✅ FFmpeg completed\n")
	return audioFile, nil
}

func transcribeToSegments(audioFile string) ([]translator.Segment, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(audioFile))
	if err != nil {
		return nil, err
	}

	f, err := os.Open(audioFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := io.Copy(part, f); err != nil {
		return nil, err
	}

	writer.WriteField("model", "faster-whisper-large-v3-fp16")
	writer.WriteField("response_format", "verbose_json")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", STTEndpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	fmt.Printf("   🔍 STT request to: %s\n", STTEndpoint)
	fmt.Printf("   🔍 Audio file: %s (size: %d)\n", audioFile, func() int64 {
		info, _ := os.Stat(audioFile)
		if info == nil {
			return 0
		}
		return info.Size()
	}())

	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	fmt.Printf("   🔍 STT response status: %s\n", resp.Status)
	fmt.Printf("   🔍 STT response headers: %v\n", resp.Header)

	var result STTResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	fmt.Printf("   🔍 STT result: text length=%d, segments=%d\n", len(result.Text), len(result.Segments))

	segments := make([]translator.Segment, len(result.Segments))
	for i, seg := range result.Segments {
		segments[i] = translator.Segment{
			Index:    i,
			Start:    seg.Start,
			End:      seg.End,
			Original: seg.Text,
		}
	}

	return segments, nil
}

type AiarkLLM struct {
	model string
	apiKey string
}

func (l *AiarkLLM) ChatCompletion(ctx context.Context, messages []translator.Message) (*translator.ChatCompletionResponse, error) {
	payload := map[string]interface{}{
		"model": l.model,
		"messages": messages,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", LLMEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	return &translator.ChatCompletionResponse{
		Content: result.Choices[0].Message.Content,
	}, nil
}

func (l *AiarkLLM) Name() string {
	return "aiark-llm"
}

func translateAll(transcript *translator.Transcript) ([]translator.Segment, error) {
	llmModel := model
	if llmModel == "" {
		llmModel = "aiark/gemma4-e2b"
	}

	llm := &AiarkLLM{model: llmModel, apiKey: apiKey}
	trans := translator.NewReflectiveTranslator(llm)
	chunker := translator.NewChunker(translator.DefaultChunkerConfig())

	chunks := chunker.Split(transcript)
	fmt.Printf("   📝 共 %d 個翻譯區塊\n", len(chunks))

	allSegments := make([]translator.Segment, 0)

	for i, chunk := range chunks {
		fmt.Printf("   📝 翻譯區塊 %d/%d...\n", i+1, len(chunks))
		chunk.TargetLang = targetLang

		if err := trans.TranslateChunk(context.Background(), chunk); err != nil {
			fmt.Printf("   ⚠️ 區塊 %d 翻譯失敗: %v\n", i+1, err)
			continue
		}

		allSegments = append(allSegments, chunk.Segments...)
	}

	return allSegments, nil
}

type AiarkTTS struct {
	apiKey      string
	outputDirAbs string
}

func (t *AiarkTTS) Synthesize(text string) (string, error) {
	payload := map[string]interface{}{
		"input":           text,
		"voice":           voice,
		"language":        language,
		"response_format": "wav",
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", TTSEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		File string `json:"file"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	downloadURL := "http://localhost:8002" + result.File
	audioPath := filepath.Join(t.outputDirAbs, fmt.Sprintf("audio_%d.wav", time.Now().UnixNano()))

	getReq, _ := http.NewRequest("GET", downloadURL, nil)
	getReq.Header.Set("Authorization", "Bearer "+t.apiKey)

	dlResp, err := client.Do(getReq)
	if err != nil {
		return "", err
	}
	defer dlResp.Body.Close()

	f, err := os.Create(audioPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, dlResp.Body)
	if err != nil {
		return "", err
	}

	return audioPath, nil
}

func burnSubtitles(videoFile, audioFile, srtFile string) (string, error) {
	absVideo, _ := filepath.Abs(videoFile)
	absSrt, _ := filepath.Abs(srtFile)
	absOutputDir, _ := filepath.Abs(outputDir)

	mergedFile := filepath.Join(absOutputDir, "final_video.mp4")

	cmd := exec.Command("ffmpeg", "-i", absVideo,
		"-vf", fmt.Sprintf("subtitles='%s':force_style='FontSize=24,PrimaryColour=&HFFFFFF&,Outline=2,Shadow=3'", absSrt),
		mergedFile, "-y")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg burn subtitles failed: %w", err)
	}

	return mergedFile, nil
}

type timedAudio struct {
	start    float64
	duration float64
	file     string
}

func synthesizeAll(segments []translator.Segment) (string, error) {
	absOutputDir, _ := filepath.Abs(outputDir)
	os.MkdirAll(absOutputDir, 0755)

	tts := &AiarkTTS{apiKey: apiKey, outputDirAbs: absOutputDir}

	var timedAudios []timedAudio

	for i, seg := range segments {
		if seg.Final == "" {
			continue
		}
		if len(seg.Final) > 200 {
			seg.Final = seg.Final[:200]
		}

		fmt.Printf("   🔊 合成 %d/%d...\n", i+1, len(segments))
		ttsPath, err := tts.Synthesize(seg.Final)
		if err != nil {
			fmt.Printf("   ⚠️ TTS 失敗: %v\n", err)
			continue
		}

		targetDuration := seg.Duration()
		adjustedFile := filepath.Join(absOutputDir, fmt.Sprintf("audio_%d.wav", i))

		if err := stretchAudio(ttsPath, adjustedFile, targetDuration); err != nil {
			fmt.Printf("   ⚠️ 調整音頻失敗: %v\n", err)
			os.Rename(ttsPath, adjustedFile)
		}

		timedAudios = append(timedAudios, timedAudio{
			start:    seg.Start,
			duration: targetDuration,
			file:     adjustedFile,
		})
	}

	if len(timedAudios) == 0 {
		return "", fmt.Errorf("no audio files generated")
	}

	dubbedFile := filepath.Join(absOutputDir, "dubbed.wav")
	if err := mergeTimedAudio(timedAudios, dubbedFile, segments); err != nil {
		return "", err
	}

	return dubbedFile, nil
}

func stretchAudio(input, output string, targetDuration float64) error {
	cmd := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", input)

	buf, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffprobe failed: %w", err)
	}

	var actualDuration float64
	fmt.Sscanf(string(buf), "%f", &actualDuration)

	if actualDuration <= 0 {
		return fmt.Errorf("invalid audio duration")
	}

	ratio := actualDuration / targetDuration
	if ratio < 0.5 {
		ratio = 0.5
	} else if ratio > 2.0 {
		ratio = 2.0
	}

	var cmdArgs []string
	if ratio != 1.0 {
		cmdArgs = []string{"-i", input, "-filter:a", fmt.Sprintf("atempo=%f", ratio), "-y", output}
	} else {
		cmdArgs = []string{"-i", input, "-y", output}
	}

	cmd = exec.Command("ffmpeg", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func mergeTimedAudio(timedAudios []timedAudio, output string, segments []translator.Segment) error {
	absOutputDir, _ := filepath.Abs(outputDir)
	padDir := filepath.Join(absOutputDir, "padded")
	os.MkdirAll(padDir, 0755)

	var paddedFiles []string
	currentTime := 0.0

	for _, ta := range timedAudios {
		taStart := ta.start
		if len(paddedFiles) > 0 {
			prevEnd := segments[len(paddedFiles)-1].End
			if taStart < prevEnd {
				taStart = prevEnd
			}
		}

		delayMs := int((taStart - currentTime) * 1000)
		if delayMs < 0 {
			delayMs = 0
		}

		paddedFile := filepath.Join(padDir, fmt.Sprintf("padded_%d.wav", len(paddedFiles)))

		var cmdArgs []string
		if delayMs > 0 {
			cmdArgs = []string{"-i", ta.file, "-af", fmt.Sprintf("apad=whole_dur=%f,adelay=%d", ta.duration+float64(delayMs)/1000, delayMs), "-y", paddedFile}
		} else {
			cmdArgs = []string{"-i", ta.file, "-y", paddedFile}
		}

		cmd := exec.Command("ffmpeg", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("   ⚠️ pad failed: %v\n", err)
			os.Rename(ta.file, paddedFile)
		}

		paddedFiles = append(paddedFiles, paddedFile)
		currentTime = ta.start + ta.duration
	}

	concatFile := filepath.Join(absOutputDir, "concat_timed.txt")
	var concatContent strings.Builder
	for _, f := range paddedFiles {
		concatContent.WriteString(fmt.Sprintf("file '%s'\n", f))
	}
	os.WriteFile(concatFile, []byte(concatContent.String()), 0644)
	defer os.Remove(concatFile)

	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatFile, "-acodec", "pcm_s16le", output, "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg merge timed audio failed: %w", err)
	}

	os.RemoveAll(padDir)
	return nil
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "檢查端點狀態",
	Run:   runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) {
	if apiKey == "" {
		apiKey = os.Getenv("LITELLM_API_KEY")
		if apiKey == "" {
			apiKey = "datasys2026"
		}
	}

	fmt.Println("🔍 檢查本地端點...")

	endpoints := []struct {
		name    string
		url     string
		checkFn func(string) error
	}{
		{"STT", STTEndpoint, checkSTT},
		{"LLM", LLMEndpoint, checkLLM},
		{"TTS", TTSEndpoint, checkTTS},
	}

	allOk := true
	for _, ep := range endpoints {
		if err := ep.checkFn(ep.url); err != nil {
			fmt.Printf("❌ %s (%s): %v\n", ep.name, ep.url, err)
			allOk = false
		} else {
			fmt.Printf("✅ %s (%s)\n", ep.name, ep.url)
		}
	}

	if allOk {
		fmt.Println("\n✅ 所有端點正常運作")
	} else {
		fmt.Println("\n❌ 部分端點異常")
		os.Exit(1)
	}
}

func checkSTT(url string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 400 {
		return nil
	}
	if resp.StatusCode >= 500 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func checkLLM(url string) error {
	payload := map[string]interface{}{
		"model": "aiark/gemma4-e2b",
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
	}
	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func checkTTS(url string) error {
	payload := map[string]interface{}{
		"input": "測試",
	}
	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

var statusCmd = &cobra.Command{
	Use:   "status [task-id]",
	Short: "顯示任務狀態",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("📋 無進行中的任務")
		} else {
			fmt.Printf("📋 任務 %s: 進行中\n", args[0])
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出所有任務",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("📋 任務清單")
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "顯示目前設定",
	Run: func(cmd *cobra.Command, args []string) {
		if apiKey == "" {
			apiKey = os.Getenv("LITELLM_API_KEY")
			if apiKey == "" {
				apiKey = "datasys2026"
			}
		}
		fmt.Println("=== KrillinAI 設定 ===")
		fmt.Printf("目標語言: %s\n", targetLang)
		fmt.Printf("輸出目錄: %s\n", outputDir)
		fmt.Printf("翻譯策略: %s\n", strategy)
		fmt.Printf("LLM 模型: %s\n", model)
		fmt.Printf("API Key: %s\n", apiKey)
		fmt.Printf("\n端點:")
		fmt.Printf("  STT: %s\n", STTEndpoint)
		fmt.Printf("  LLM: %s\n", LLMEndpoint)
		fmt.Printf("  TTS: %s\n", TTSEndpoint)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細輸出")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API Key (或 LITELLM_API_KEY)")
	runCmd.Flags().StringVarP(&targetLang, "target-lang", "t", "繁體中文", "目標語言")
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "輸出目錄")
	runCmd.Flags().StringVarP(&strategy, "strategy", "s", "reflective", "翻譯策略 (fast/reflective)")
	runCmd.Flags().StringVarP(&model, "model", "m", "aiark/gemma4-e2b", "指定 LLM 模型")
	runCmd.Flags().StringVar(&voice, "voice", "Alex", "TTS 語音")
	runCmd.Flags().StringVar(&language, "lang", "Chinese", "TTS 語言")
	runCmd.Flags().StringVar(&ttsModel, "tts-model", "Qwen3-TTS-0.6B", "TTS 模型")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "執行失敗: %v\n", err)
		os.Exit(1)
	}
}