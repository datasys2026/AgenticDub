package openai

import (
	"github.com/sashabaranov/go-openai"
	"krillin-ai/config"
)

type Client struct {
	client *openai.Client
}

func NewClient() *Client {
	client := openai.NewClient(config.Conf.Openai.ApiKey)
	return &Client{client: client}
}
