package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

var (
	ErrInvalidJSONResponse = errors.New("invalid JSON response from LLM")
	ErrEmptyResponse       = errors.New("empty response from LLM")
)

type chatResponse struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

func newChatRequest(ctx context.Context, url, apiKey string, jsonData []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func doChatRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, newHTTPError(resp.StatusCode, string(body))
	}

	return body, nil
}

func parseChatResponse(body []byte) (*chatResponse, error) {
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if resp.Error.Message != "" {
		return nil, errors.New(resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return nil, ErrEmptyResponse
	}

	return &chatResponse{
		Content: resp.Choices[0].Message.Content,
	}, nil
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return e.Message
}

func newHTTPError(statusCode int, body string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    body,
	}
}