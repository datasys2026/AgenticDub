package service

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

	"krillin-ai/internal/providers/llm"
	"krillin-ai/internal/translator"
)

const (
	defaultCLIPipelineSTTEndpoint = "http://localhost:8006/v1/audio/transcriptions"
	defaultCLIPipelineLLMEndpoint = "http://localhost:4000/v1/chat/completions"
	defaultCLIPipelineTTSEndpoint = "http://localhost:8002/v1/audio/speech"
	defaultCLIPipelineLLMModel    = "aiark/gemma4-e2b"
)

type LegacyCLIPipelineOptions struct {
	Input       string
	TargetLang  string
	OutputDir   string
	Strategy    string
	Model       string
	APIKey      string
	Voice       string
	Lang        string
	STTEndpoint string
	LLMEndpoint string
	TTSEndpoint string
}

type LegacyCLIPipelineResult struct {
	SRTFile     string
	DubbedAudio string
	MergedVideo string
	Segments    []translator.Segment
}

type cliSTTResult struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

type cliLLM struct {
	model    string
	apiKey   string
	endpoint string
}

func (c *cliLLM) ChatCompletion(ctx context.Context, messages []llm.Message) (*llm.ChatCompletionResponse, error) {
	payload := map[string]interface{}{
		"model":    c.model,
		"messages": messages,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

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

	return &llm.ChatCompletionResponse{Content: result.Choices[0].Message.Content}, nil
}

func (c *cliLLM) Name() string {
	return "aiark-llm"
}

type cliTTS struct {
	apiKey    string
	voice     string
	lang      string
	endpoint  string
	outputDir string
}

func (t *cliTTS) Synthesize(ctx context.Context, text string) (string, error) {
	if len(strings.TrimSpace(text)) == 0 {
		return "", fmt.Errorf("empty text")
	}

	payload := map[string]interface{}{
		"input":           text,
		"voice":           t.voice,
		"language":        t.lang,
		"response_format": "wav",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, bytes.NewBuffer(jsonData))
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
	if result.File == "" {
		return "", fmt.Errorf("tts response missing file url")
	}

	downloadURL := result.File
	if !strings.HasPrefix(downloadURL, "http://") && !strings.HasPrefix(downloadURL, "https://") {
		base := strings.TrimSuffix(t.endpoint, "/v1/audio/speech")
		base = strings.TrimSuffix(base, "/")
		if !strings.HasPrefix(downloadURL, "/") {
			downloadURL = "/" + downloadURL
		}
		downloadURL = base + downloadURL
	}

	audioPath := filepath.Join(t.outputDir, fmt.Sprintf("audio_%d.wav", time.Now().UnixNano()))
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
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

	if _, err := io.Copy(f, dlResp.Body); err != nil {
		return "", err
	}

	return audioPath, nil
}

func RunLegacyCLIPipeline(ctx context.Context, opts LegacyCLIPipelineOptions) (*LegacyCLIPipelineResult, error) {
	if strings.TrimSpace(opts.Input) == "" {
		return nil, fmt.Errorf("input is required")
	}
	if opts.TargetLang == "" {
		opts.TargetLang = "繁體中文"
	}
	if opts.Model == "" {
		opts.Model = defaultCLIPipelineLLMModel
	}
	if opts.STTEndpoint == "" {
		opts.STTEndpoint = defaultCLIPipelineSTTEndpoint
	}
	if opts.LLMEndpoint == "" {
		opts.LLMEndpoint = defaultCLIPipelineLLMEndpoint
	}
	if opts.TTSEndpoint == "" {
		opts.TTSEndpoint = defaultCLIPipelineTTSEndpoint
	}
	if opts.OutputDir == "" {
		opts.OutputDir = "./output"
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir failed: %w", err)
	}

	audioFile, err := extractAudioLegacy(opts.Input, opts.OutputDir)
	if err != nil {
		return nil, err
	}
	defer os.Remove(audioFile)

	segments, err := transcribeToSegmentsLegacy(ctx, opts.STTEndpoint, audioFile)
	if err != nil {
		return nil, fmt.Errorf("stt failed: %w", err)
	}

	transcript := &translator.Transcript{Segments: segments, Language: "en"}
	translatedSegments, err := translateAllLegacy(ctx, opts.Model, opts.APIKey, opts.LLMEndpoint, opts.TargetLang, opts.Strategy, transcript)
	if err != nil {
		return nil, fmt.Errorf("translation failed: %w", err)
	}

	absOutputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output dir failed: %w", err)
	}

	srtFile := filepath.Join(absOutputDir, "subtitles.srt")
	gen := translator.NewSRTGenerator()
	srtContent := gen.Generate(translatedSegments)
	if err := os.WriteFile(srtFile, []byte(srtContent), 0644); err != nil {
		return nil, fmt.Errorf("write srt failed: %w", err)
	}

	dubbedFile, err := synthesizeAllLegacy(ctx, opts.OutputDir, opts.Lang, opts.Voice, opts.TTSEndpoint, opts.APIKey, translatedSegments)
	if err != nil {
		return nil, fmt.Errorf("tts failed: %w", err)
	}

	mergedFile, err := burnSubtitlesLegacy(opts.Input, dubbedFile, srtFile, opts.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("burn subtitles failed: %w", err)
	}

	return &LegacyCLIPipelineResult{
		SRTFile:     srtFile,
		DubbedAudio: dubbedFile,
		MergedVideo: mergedFile,
		Segments:    translatedSegments,
	}, nil
}

func extractAudioLegacy(input, outputDir string) (string, error) {
	absInput, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}

	audioFile := filepath.Join(absOutputDir, "audio.wav")

	var cmd *exec.Cmd
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		videoFile := filepath.Join(absOutputDir, "video.mp4")
		cmd = exec.Command("yt-dlp", "-f", "best[ext=mp4]/best", "-o", videoFile, input)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("yt-dlp failed: %w", err)
		}
		defer os.Remove(videoFile)

		cmd = exec.Command("ffmpeg", "-i", videoFile, "-vn", "-acodec", "pcm_s16le", "-ac", "1", "-ar", "16000", audioFile, "-y")
	} else {
		cmd = exec.Command("ffmpeg", "-i", absInput, "-vn", "-acodec", "pcm_s16le", "-ac", "1", "-ar", "16000", audioFile, "-y")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	return audioFile, nil
}

func transcribeToSegmentsLegacy(ctx context.Context, endpoint, audioFile string) ([]translator.Segment, error) {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
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

	var result cliSTTResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

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

func translateAllLegacy(ctx context.Context, model, apiKey, llmEndpoint, targetLang, strategy string, transcript *translator.Transcript) ([]translator.Segment, error) {
	if strategy == "" {
		strategy = "reflective"
	}

	_ = strategy // keep current CLI behavior aligned: same reflective pipeline.

	provider := &cliLLM{
		model:    model,
		apiKey:   apiKey,
		endpoint: llmEndpoint,
	}
	if provider.model == "" {
		provider.model = defaultCLIPipelineLLMModel
	}
	if provider.endpoint == "" {
		provider.endpoint = defaultCLIPipelineLLMEndpoint
	}

	trans := translator.NewReflectiveTranslator(provider)
	chunker := translator.NewChunker(translator.DefaultChunkerConfig())

	chunks := chunker.Split(transcript)
	allSegments := make([]translator.Segment, 0)
	for _, chunk := range chunks {
		chunk.TargetLang = targetLang

		if err := trans.TranslateChunk(ctx, chunk); err != nil {
			// keep flow tolerant like the existing CLI behavior.
			continue
		}

		allSegments = append(allSegments, chunk.Segments...)
	}

	return allSegments, nil
}

func synthesizeAllLegacy(ctx context.Context, outputDir, language, voice, endpoint, apiKey string, segments []translator.Segment) (string, error) {
	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		return "", err
	}

	tts := &cliTTS{
		apiKey:    apiKey,
		voice:     voice,
		lang:      language,
		endpoint:  endpoint,
		outputDir: absOutputDir,
	}
	if tts.endpoint == "" {
		tts.endpoint = defaultCLIPipelineTTSEndpoint
	}

	var audioFiles []string
	for _, seg := range segments {
		if seg.Final == "" {
			continue
		}
		text := seg.Final
		if len(text) > 200 {
			text = text[:200]
		}

		path, err := tts.Synthesize(ctx, text)
		if err != nil {
			continue
		}
		audioFiles = append(audioFiles, path)
	}

	if len(audioFiles) == 0 {
		return "", fmt.Errorf("no audio files generated")
	}

	dubbedFile := filepath.Join(absOutputDir, "dubbed.wav")
	if len(audioFiles) == 1 {
		if err := os.Rename(audioFiles[0], dubbedFile); err != nil {
			return "", err
		}
		return dubbedFile, nil
	}

	concatFile := filepath.Join(absOutputDir, "concat.txt")
	var concatContent strings.Builder
	for _, f := range audioFiles {
		concatContent.WriteString(fmt.Sprintf("file '%s'\n", f))
	}
	if err := os.WriteFile(concatFile, []byte(concatContent.String()), 0644); err != nil {
		return "", err
	}
	defer os.Remove(concatFile)

	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatFile, "-acodec", "pcm_s16le", dubbedFile, "-y")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg merge failed: %w", err)
	}

	for _, f := range audioFiles {
		_ = os.Remove(f)
	}

	return dubbedFile, nil
}

func burnSubtitlesLegacy(input, audioFile, srtFile, outputDir string) (string, error) {
	absVideo, _ := filepath.Abs(input)
	absSrt, _ := filepath.Abs(srtFile)
	absOutputDir, _ := filepath.Abs(outputDir)

	_ = audioFile
	mergedFile := filepath.Join(absOutputDir, "final_video.mp4")

	cmd := exec.Command(
		"ffmpeg",
		"-i", absVideo,
		"-vf", fmt.Sprintf("subtitles='%s':force_style='FontSize=24,PrimaryColour=&HFFFFFF&,Outline=2,Shadow=3'", absSrt),
		mergedFile, "-y",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg burn subtitles failed: %w", err)
	}

	return mergedFile, nil
}
