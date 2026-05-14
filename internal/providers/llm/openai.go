package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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

func buildChatURL(baseURL string) string {
	base := strings.TrimSuffix(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func buildHTTPClient(proxyAddr string) *http.Client {
	client := &http.Client{Timeout: 60 * time.Second}
	if proxyAddr == "" {
		return client
	}
	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return client
	}
	client.Transport = &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	return client
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	url := buildChatURL(p.baseURL)

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

	client := buildHTTPClient(p.proxyAddr)

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
