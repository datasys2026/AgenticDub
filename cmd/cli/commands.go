package main

import (
	"bytes"
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
)

var (
	targetLang string
	outputDir string
	model     string
	strategy  string
	verbose   bool
	apiKey    string
)

const (
	STTEndpoint = "http://localhost:8006/v1/audio/transcriptions"
	LLMEndpoint = "http://localhost:4000/v1/chat/completions"
	TTSEndpoint = "http://localhost:8002/v1/audio/speech"
)

var rootCmd = &cobra.Command{
	Use:   "krilin-ai",
	Short: "AI 影片翻譯配音工具",
	Long:  `KrillinAI - 影片翻譯配音工具

端點配置:
- STT:  localhost:8006 (faster-whisper)
- LLM:  localhost:4000 (LiteLLM)
- TTS:  localhost:8002 (Qwen3-TTS)`,
}

var runCmd = &cobra.Command{
	Use:   "run <input>",
	Short: "執行完整翻譯流程",
	Long: `執行影片翻譯配音流程。
	輸入可以是本地檔案路徑或 YouTube URL。`,
	Args: cobra.ExactArgs(1),
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

	os.MkdirAll(outputDir, 0755)

	audioFile, err := extractAudio(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 音頻提取失敗: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(audioFile)
	fmt.Printf("✅ 音頻已提取: %s\n", audioFile)

	fmt.Printf("🔄 正在進行語音辨識 (STT)...\n")
	transcript, err := transcribe(audioFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ STT 失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ STT 完成 (%.1f 秒, 語言: %s)\n", transcript.Duration, transcript.Language)

	fmt.Printf("🔄 正在翻譯字幕...\n")
	translated, err := translate(transcript.Text)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 翻譯失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 翻譯完成\n")

	fmt.Printf("🔄 正在生成語音 (TTS)...\n")
	audioPath, err := synthesize(translated)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ TTS 失敗: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ 語音生成完成: %s\n", audioPath)

	fmt.Printf("\n✅ 流程完成!\n")
	fmt.Printf("   字幕檔: %s/subtitles.srt\n", outputDir)
	fmt.Printf("   配音檔: %s\n", audioPath)
}

type Transcript struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
}

func extractAudio(input string) (string, error) {
	audioFile := filepath.Join(outputDir, "audio.wav")

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		fmt.Printf("   📥 下載影片...\n")
		cmd := exec.Command("yt-dlp", "-f", "best[ext=mp4]/best",
			"--extract-audio", "--audio-format", "wav",
			"--audio-quality", "0", "-o", filepath.Join(outputDir, "video.mp4"), input)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("yt-dlp failed: %w", err)
		}
		videoFile := filepath.Join(outputDir, "video.mp4")
		defer os.Remove(videoFile)
	}

	cmd := exec.Command("ffmpeg", "-i", input,
		"-vn", "-acodec", "pcm_s16le", "-ac", "1", "-ar", "16000",
		audioFile, "-y")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	return audioFile, nil
}

func transcribe(audioFile string) (*Transcript, error) {
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

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", STTEndpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Text            string  `json:"text"`
		Language        string  `json:"language"`
		Duration        float64 `json:"duration"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Transcript{
		Text:     result.Text,
		Language: result.Language,
		Duration: result.Duration,
	}, nil
}

func translate(text string) (string, error) {
	llmModel := model
	if llmModel == "" {
		llmModel = "aiark/gemma4-e2b"
	}

	systemPrompt := fmt.Sprintf(`你是一個專業的字幕翻譯。將以下字幕翻譯成%s。
規則：
- 保持自然、口語化的表達
- 每行字幕不超過42個字
- 考慮說話的語境和語氣
- 不要翻譯專有名詞，保留原文`, targetLang)

	payload := map[string]interface{}{
		"model": llmModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": text},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", LLMEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
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
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no translation result")
	}

	return result.Choices[0].Message.Content, nil
}

type TTSResponse struct {
	File       string `json:"file"`
	SampleRate int    `json:"sample_rate"`
}

func synthesize(text string) (string, error) {
	payload := map[string]interface{}{
		"input": text,
	}

	jsonData, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", TTSEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result TTSResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	audioPath := filepath.Join(outputDir, "dubbed.wav")
	downloadURL := "http://localhost:8002" + result.File

	getReq, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", err
	}
	getReq.Header.Set("Authorization", "Bearer "+apiKey)

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
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "執行失敗: %v\n", err)
		os.Exit(1)
	}
}