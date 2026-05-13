package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type OpenAIProvider struct {
	baseURL   string
	apiKey    string
	model     string
	proxyAddr string
}

func NewOpenAIProvider(baseURL, apiKey, model, proxyAddr string) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		proxyAddr: proxyAddr,
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	url := p.baseURL + "/chat/completions"
	if !strings.HasSuffix(url, "/v1") && !strings.HasSuffix(url, "/v1/") {
		url = strings.TrimSuffix(url, "/") + "/v1/chat/completions"
	}

	reqBody := map[string]any{
		"model":    p.model,
		"messages": messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	if p.proxyAddr != "" {
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(nil),
		}
	}

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
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error.Message != "" {
		return nil, &LLMError{Message: result.Error.Message}
	}

	if len(result.Choices) == 0 {
		return nil, ErrEmptyResponse
	}

	return &ChatCompletionResponse{
		Content: result.Choices[0].Message.Content,
	}, nil
}

type LLMError struct {
	Message string
}

func (e *LLMError) Error() string {
	return e.Message
}
