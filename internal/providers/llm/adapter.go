package llm

import "context"

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
