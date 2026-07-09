package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	xaiauth "krillin-ai/internal/auth/xai"
	"krillin-ai/internal/providers/llm"
	"krillin-ai/internal/service"
)

var (
	targetLang    string
	outputDir     string
	model         string
	strategy      string
	verbose       bool
	apiKey        string
	voice         string
	language      string
	ttsModel      string
	xaiTokenPath  string
	xaiBaseURL    string
	xaiProbeModel string
)

const (
	STTEndpoint = "http://localhost:8006/v1/audio/transcriptions"
	LLMEndpoint = "http://localhost:4000/v1/chat/completions"
	TTSEndpoint = "http://localhost:8002/v1/audio/speech"
)

var rootCmd = &cobra.Command{
	Use:   "agenticdub",
	Short: "AI 影片翻譯配音工具",
	Long:  `AgenticDub - 影片翻譯配音工具`,
}

var runCmd = &cobra.Command{
	Use:   "run <input>",
	Short: "執行完整翻譯流程",
	Long:  `執行影片翻譯配音流程。`,
	Args:  cobra.ExactArgs(1),
	Run:   runVideo,
}

func runVideo(cmd *cobra.Command, args []string) {
	input := args[0]

	if apiKey == "" {
		apiKey = os.Getenv("LITELLM_API_KEY")
		if apiKey == "" {
			apiKey = "datasys2026"
		}
	}
	if outputDir == "" {
		outputDir = "./output"
	}
	if model == "" {
		model = "aiark/gemma4-e2b"
	}

	fmt.Printf("🎬 開始處理影片: %s\n", input)
	fmt.Printf("   目標語言: %s\n", targetLang)
	fmt.Printf("   輸出目錄: %s\n", outputDir)
	fmt.Printf("   翻譯策略: %s\n", strategy)
	fmt.Printf("   TTS 模型: %s (%s - %s)\n", ttsModel, voice, language)

	res, err := service.RunLegacyCLIPipeline(cmd.Context(), service.LegacyCLIPipelineOptions{
		Input:       input,
		TargetLang:  targetLang,
		OutputDir:   outputDir,
		Strategy:    strategy,
		Model:       model,
		APIKey:      apiKey,
		Voice:       voice,
		Lang:        language,
		STTEndpoint: STTEndpoint,
		LLMEndpoint: LLMEndpoint,
		TTSEndpoint: TTSEndpoint,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ 流程失敗: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 流程完成!\n")
	fmt.Printf("   字幕檔: %s\n", res.SRTFile)
	fmt.Printf("   配音檔: %s\n", res.DubbedAudio)
	fmt.Printf("   影片檔: %s\n", res.MergedVideo)
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
		fmt.Println("=== AgenticDub 設定 ===")
		fmt.Printf("目標語言: %s\n", targetLang)
		fmt.Printf("輸出目錄: %s\n", outputDir)
		fmt.Printf("翻譯策略: %s\n", strategy)
		fmt.Printf("LLM 模型: %s\n", model)
		fmt.Printf("API Key: %s\n", apiKey)
		fmt.Printf("\n端點:\n")
		fmt.Printf("  STT: %s\n", STTEndpoint)
		fmt.Printf("  LLM: %s\n", LLMEndpoint)
		fmt.Printf("  TTS: %s\n", TTSEndpoint)
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "管理第三方登入狀態",
}

var xaiAuthCmd = &cobra.Command{
	Use:   "xai",
	Short: "管理 xAI / Grok OAuth 狀態",
}

var xaiAuthStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "檢查 xAI / Grok OAuth token",
	Run:   runXAIAuthStatus,
}

var xaiAuthProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "使用 xAI / Grok OAuth token 呼叫 /v1/responses",
	Run:   runXAIAuthProbe,
}

func resolveXAITokenPath(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if envValue := os.Getenv("XAI_OAUTH_TOKEN_PATH"); envValue != "" {
		return envValue
	}
	return xaiauth.DefaultTokenPath()
}

type xaiProbeOptions struct {
	TokenPath string
	BaseURL   string
	Model     string
	Client    llm.HTTPDoer
	Stdout    io.Writer
}

func runXAIProbe(ctx context.Context, opts xaiProbeOptions) error {
	if opts.TokenPath == "" {
		opts.TokenPath = xaiauth.DefaultTokenPath()
	}
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.x.ai/v1"
	}
	if opts.Model == "" {
		opts.Model = "grok-4.20-0309-non-reasoning"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}

	store := xaiauth.NewFileTokenStore(opts.TokenPath)
	provider := llm.NewXAIOAuthProvider(opts.BaseURL, opts.Model, xaiauth.NewFileTokenSource(store), opts.Client)
	resp, err := provider.ChatCompletion(ctx, []llm.Message{
		{Role: "user", Content: "Reply with exactly: agenticdub-probe-ok"},
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "xAI OAuth probe: ok\n")
	fmt.Fprintf(opts.Stdout, "Model: %s\n", opts.Model)
	fmt.Fprintf(opts.Stdout, "Response: %s\n", strings.TrimSpace(resp.Content))
	return nil
}

func runXAIAuthProbe(cmd *cobra.Command, args []string) {
	path := resolveXAITokenPath(xaiTokenPath)
	if err := runXAIProbe(cmd.Context(), xaiProbeOptions{
		TokenPath: path,
		BaseURL:   xaiBaseURL,
		Model:     xaiProbeModel,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "xAI OAuth probe failed: %v\n", err)
		os.Exit(1)
	}
}

func runXAIAuthStatus(cmd *cobra.Command, args []string) {
	path := resolveXAITokenPath(xaiTokenPath)
	store := xaiauth.NewFileTokenStore(path)
	token, err := store.Load()
	if err != nil {
		if errors.Is(err, xaiauth.ErrTokenNotFound) {
			fmt.Fprintf(os.Stderr, "xAI OAuth token not found: %s\n", path)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "xAI OAuth token check failed: %v\n", err)
		os.Exit(1)
	}

	if _, err := xaiauth.NewFileTokenSource(store).BearerToken(cmd.Context()); err != nil {
		fmt.Fprintf(os.Stderr, "xAI OAuth token invalid: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("xAI OAuth token: ok\n")
	fmt.Printf("Token path: %s\n", path)
	if !token.ExpiresAt.IsZero() {
		fmt.Printf("Expires at: %s\n", token.ExpiresAt.Format(time.RFC3339))
	}
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(xaiAuthCmd)
	xaiAuthCmd.AddCommand(xaiAuthStatusCmd)
	xaiAuthCmd.AddCommand(xaiAuthProbeCmd)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細輸出")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API Key (或 LITELLM_API_KEY)")
	runCmd.Flags().StringVarP(&targetLang, "target-lang", "t", "繁體中文", "目標語言")
	runCmd.Flags().StringVarP(&outputDir, "output", "o", "./output", "輸出目錄")
	runCmd.Flags().StringVarP(&strategy, "strategy", "s", "reflective", "翻譯策略 (fast/reflective)")
	runCmd.Flags().StringVarP(&model, "model", "m", "aiark/gemma4-e2b", "指定 LLM 模型")
	runCmd.Flags().StringVar(&voice, "voice", "Alex", "TTS 語音")
	runCmd.Flags().StringVar(&language, "lang", "Chinese", "TTS 語言")
	runCmd.Flags().StringVar(&ttsModel, "tts-model", "Qwen3-TTS-0.6B", "TTS 模型")
	xaiAuthStatusCmd.Flags().StringVar(&xaiTokenPath, "token-path", "", "xAI OAuth token file path")
	xaiAuthProbeCmd.Flags().StringVar(&xaiTokenPath, "token-path", "", "xAI OAuth token file path")
	xaiAuthProbeCmd.Flags().StringVar(&xaiBaseURL, "base-url", "https://api.x.ai/v1", "xAI API base URL")
	xaiAuthProbeCmd.Flags().StringVar(&xaiProbeModel, "model", "grok-4.20-0309-non-reasoning", "xAI model to probe")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "執行失敗: %v\n", err)
		os.Exit(1)
	}
}
