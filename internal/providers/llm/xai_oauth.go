package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	xaiauth "krillin-ai/internal/auth/xai"
)

type BearerTokenSource interface {
	BearerToken(ctx context.Context) (string, error)
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type XAIOAuthProvider struct {
	baseURL     string
	model       string
	tokenSource BearerTokenSource
	client      HTTPDoer
}

func NewXAIOAuthProvider(baseURL, model string, tokenSource BearerTokenSource, client HTTPDoer) *XAIOAuthProvider {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &XAIOAuthProvider{
		baseURL:     baseURL,
		model:       model,
		tokenSource: tokenSource,
		client:      client,
	}
}

func NewXAIOAuthProviderFromTokenFile(baseURL, model, tokenPath string) *XAIOAuthProvider {
	if tokenPath == "" {
		tokenPath = xaiauth.DefaultTokenPath()
	}
	store := xaiauth.NewFileTokenStore(tokenPath)
	return NewXAIOAuthProvider(baseURL, model, xaiauth.NewFileTokenSource(store), nil)
}

func (p *XAIOAuthProvider) Name() string {
	return "xai-oauth"
}

func buildXAIResponsesURL(baseURL string) string {
	base := strings.TrimSuffix(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/responses"
	}
	return base + "/v1/responses"
}

func (p *XAIOAuthProvider) ChatCompletion(ctx context.Context, messages []Message) (*ChatCompletionResponse, error) {
	token, err := p.tokenSource.BearerToken(ctx)
	if err != nil {
		return nil, err
	}

	reqBody := map[string]any{
		"model": p.model,
		"input": messages,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildXAIResponsesURL(p.baseURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result, err := parseXAIResponses(resp)
	if err != nil {
		return nil, err
	}
	return &ChatCompletionResponse{Content: result}, nil
}

func parseXAIResponses(resp *http.Response) (string, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		message := parseXAIErrorMessage(data)
		if resp.StatusCode == http.StatusForbidden {
			return "", &XAIEntitlementError{StatusCode: resp.StatusCode, Message: message}
		}
		return "", &LLMError{Message: fmt.Sprintf("xAI OAuth request failed with HTTP %d: %s", resp.StatusCode, message)}
	}

	var result xaiResponsesResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if result.OutputText != "" {
		return result.OutputText, nil
	}
	for _, output := range result.Output {
		for _, content := range output.Content {
			if content.Text != "" {
				return content.Text, nil
			}
		}
	}
	return "", ErrEmptyResponse
}

type xaiResponsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func parseXAIErrorMessage(data []byte) string {
	var result struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &result); err == nil {
		if result.Error.Message != "" {
			return result.Error.Message
		}
		if result.Error.Code != "" {
			return result.Error.Code
		}
	}
	return strings.TrimSpace(string(data))
}

type XAIEntitlementError struct {
	StatusCode int
	Message    string
}

func (e *XAIEntitlementError) Error() string {
	if e.Message == "" {
		return "xAI OAuth request was forbidden; subscription tier or OAuth entitlement may not allow this model/API surface"
	}
	return fmt.Sprintf("xAI OAuth request was forbidden: %s (check subscription tier or OAuth entitlement)", e.Message)
}
