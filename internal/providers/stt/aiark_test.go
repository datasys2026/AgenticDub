package stt

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

var (
	ErrAudioFileNotFound = errors.New("audio file not found")
	ErrTranscribeFailed  = errors.New("transcription failed")
)

func TestSTTProviderInterface(t *testing.T) {
	t.Run("STTProvider interface must implement Transcribe", func(t *testing.T) {
		var provider STTProvider = &MockSTTProvider{}

		ctx := context.Background()
		transcript, err := provider.Transcribe(ctx, "test.wav", "en")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(transcript.Segments) == 0 {
			t.Fatal("expected at least one segment")
		}
	})

	t.Run("STTProvider must return name", func(t *testing.T) {
		provider := &MockSTTProvider{}
		if provider.Name() == "" {
			t.Error("expected non-empty name")
		}
	})
}

type MockSTTProvider struct{}

func (m *MockSTTProvider) Transcribe(ctx context.Context, audioPath string, language string) (*Transcript, error) {
	return &Transcript{
		Segments: []TranscriptSegment{
			{Start: 0.0, End: 2.5, Text: "Hello world"},
		},
		Language: language,
	}, nil
}

func (m *MockSTTProvider) Name() string {
	return "mock-stt"
}

func TestAiarkSTTProvider(t *testing.T) {
	t.Run("AiarkSTTProvider implements STTProvider", func(t *testing.T) {
		var provider STTProvider = NewAiarkSTTProvider("http://localhost:4000", "test-key")
		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
	})
}

type AiarkSTTProvider struct {
	baseURL string
	apiKey  string
	client  *openai.Client
}

func NewAiarkSTTProvider(baseURL, apiKey string) *AiarkSTTProvider {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL + "/v1"
	return &AiarkSTTProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  openai.NewClientWithConfig(cfg),
	}
}

func (p *AiarkSTTProvider) Transcribe(ctx context.Context, audioPath string, language string) (*Transcript, error) {
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return nil, ErrAudioFileNotFound
	}

	f, err := os.Open(audioPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	resp, err := p.client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    "faster-whisper-large-v3-fp16",
		FilePath: audioPath,
		Reader:   f,
		Format:   openai.AudioResponseFormatVerboseJSON,
		Language: language,
	})
	if err != nil {
		return nil, ErrTranscribeFailed
	}

	segments := make([]TranscriptSegment, 0, len(resp.Segments))
	for _, seg := range resp.Segments {
		segments = append(segments, TranscriptSegment{
			Start: seg.Start,
			End:   seg.End,
			Text:  seg.Text,
		})
	}

	return &Transcript{
		Segments: segments,
		Language: resp.Language,
	}, nil
}

func (p *AiarkSTTProvider) Name() string {
	return "aiark-stt"
}