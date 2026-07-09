package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"krillin-ai/internal/types"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BearerTokenSource interface {
	BearerToken(ctx context.Context) (string, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type XAIOAuthProvider struct {
	baseURL     string
	tokenSource BearerTokenSource
	client      HTTPDoer
}

func NewXAIOAuthProvider(baseURL string, tokenSource BearerTokenSource, client HTTPDoer) *XAIOAuthProvider {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &XAIOAuthProvider{
		baseURL:     baseURL,
		tokenSource: tokenSource,
		client:      client,
	}
}

func (p *XAIOAuthProvider) Transcription(audioFile, language, workDir string) (*types.TranscriptionData, error) {
	return p.TranscriptionWithContext(context.Background(), audioFile, language, workDir)
}

func (p *XAIOAuthProvider) TranscriptionWithContext(ctx context.Context, audioFile, language, workDir string) (*types.TranscriptionData, error) {
	token, err := p.tokenSource.BearerToken(ctx)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if language != "" {
		if err := writer.WriteField("language", language); err != nil {
			return nil, err
		}
	}
	if err := writer.WriteField("format", "true"); err != nil {
		return nil, err
	}

	file, err := os.Open(audioFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(audioFile))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildXAISTTURL(p.baseURL), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("xAI STT request failed with HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var result xaiSTTResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result.toTranscriptionData(), nil
}

func buildXAISTTURL(baseURL string) string {
	base := strings.TrimSuffix(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/stt"
	}
	return base + "/v1/stt"
}

type xaiSTTResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
	Words    []struct {
		Text  string  `json:"text"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"words"`
}

func (r xaiSTTResponse) toTranscriptionData() *types.TranscriptionData {
	words := make([]types.Word, 0, len(r.Words))
	for i, word := range r.Words {
		words = append(words, types.Word{
			Num:   i,
			Text:  word.Text,
			Start: word.Start,
			End:   word.End,
		})
	}

	segments := []types.TranscriptionSegment{}
	if r.Text != "" {
		end := r.Duration
		if end == 0 && len(words) > 0 {
			end = words[len(words)-1].End
		}
		segments = append(segments, types.TranscriptionSegment{
			Start: 0,
			End:   end,
			Text:  r.Text,
		})
	}

	return &types.TranscriptionData{
		Language: r.Language,
		Text:     r.Text,
		Words:    words,
		Segments: segments,
	}
}
