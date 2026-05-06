package whisper

import (
	"github.com/sashabaranov/go-openai"
	"krillin-ai/config"
	"net/http"
)

type Client struct {
	client *openai.Client
	model  string
}

func NewClient(baseUrl, apiKey, model, proxyAddr string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	if baseUrl != "" {
		cfg.BaseURL = baseUrl
	}

	if proxyAddr != "" {
		transport := &http.Transport{
			Proxy: http.ProxyURL(config.Conf.App.ParsedProxy),
		}
		cfg.HTTPClient = &http.Client{
			Transport: transport,
		}
	}

	client := openai.NewClientWithConfig(cfg)
	return &Client{client: client, model: model}
}
