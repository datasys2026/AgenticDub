package llm

import (
	"context"
	"encoding/json"
	"strings"
)

type AiarkLLMProvider struct {
	baseURL  string
	apiKey   string
	model    string
	endpoint string
}

func NewAiarkLLMProvider(baseURL, apiKey, model string) *AiarkLLMProvider {
	return &AiarkLLMProvider{
		baseURL:  baseURL,
		apiKey:   apiKey,
		model:    model,
		endpoint: "/v1/chat/completions",
	}
}

func (p *AiarkLLMProvider) Name() string {
	return "aiark-llm"
}

func (p *AiarkLLMProvider) ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": messages,
	}

	if strings.Contains(p.model, "deepseek") || strings.Contains(p.model, "gemma4:26b") {
		reqBody["think"] = false
	}

	resp, err := p.makeRequest(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	return &ChatCompletionResponse{
		Content: resp.Content,
	}, nil
}

func (p *AiarkLLMProvider) makeRequest(ctx context.Context, body map[string]interface{}) (*chatResponse, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := newChatRequest(ctx, p.baseURL+p.endpoint, p.apiKey, jsonData)
	if err != nil {
		return nil, err
	}

	resp, err := doChatRequest(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return parseChatResponse(resp)
}

type ChatCompleterAdapter struct {
	provider LLMProvider
}

func NewChatCompleterAdapter(provider LLMProvider) *ChatCompleterAdapter {
	return &ChatCompleterAdapter{provider: provider}
}

func (a *ChatCompleterAdapter) ChatCompletion(query string) (string, error) {
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: query}}
	resp, err := a.provider.ChatCompletion(ctx, messages)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}