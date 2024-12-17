package service

import "krillin-ai/pkg/openai"

type Service struct {
	OpenaiClient *openai.Client
}

func NewService() *Service {
	return &Service{
		OpenaiClient: openai.NewClient(),
	}
}
