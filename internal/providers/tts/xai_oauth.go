package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DefaultXAILanguage = "auto"
	DefaultXAIVoice    = "eve"
)

type BearerTokenSource interface {
	BearerToken(ctx context.Context) (string, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type XAIOAuthClient struct {
	baseURL     string
	tokenSource BearerTokenSource
	client      HTTPDoer
}

func NewXAIOAuthClient(baseURL string, tokenSource BearerTokenSource, client HTTPDoer) *XAIOAuthClient {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &XAIOAuthClient{
		baseURL:     baseURL,
		tokenSource: tokenSource,
		client:      client,
	}
}

func (c *XAIOAuthClient) Text2Speech(text string, voice string, outputFile string) error {
	return c.Text2SpeechWithContext(context.Background(), text, voice, outputFile)
}

func (c *XAIOAuthClient) Text2SpeechWithContext(ctx context.Context, text string, voice string, outputFile string) error {
	if strings.TrimSpace(text) == "" {
		return ErrEmptyText
	}
	if voice == "" {
		voice = DefaultXAIVoice
	}

	token, err := c.tokenSource.BearerToken(ctx)
	if err != nil {
		return err
	}

	reqBody, err := json.Marshal(map[string]string{
		"text":     text,
		"voice_id": voice,
		"language": DefaultXAILanguage,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildXAITTSURL(c.baseURL), bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("xAI TTS request failed with HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	out, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func buildXAITTSURL(baseURL string) string {
	base := strings.TrimSuffix(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/tts"
	}
	return base + "/v1/tts"
}
