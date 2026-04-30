package tts

import (
	"context"
	"testing"
)

func TestTTSProviderInterface(t *testing.T) {
	t.Run("TTSProvider must implement Synthesize", func(t *testing.T) {
		var provider TTSProvider = &MockTTSProvider{}

		ctx := context.Background()
		result, err := provider.Synthesize(ctx, "你好世界", "default")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result.Data) == 0 {
			t.Fatal("expected non-empty audio data")
		}
	})

	t.Run("TTSProvider must return name", func(t *testing.T) {
		provider := &MockTTSProvider{}
		if provider.Name() == "" {
			t.Error("expected non-empty name")
		}
	})

	t.Run("TTSProvider must implement SynthesizeBatch", func(t *testing.T) {
		var provider TTSProvider = &MockTTSProvider{}

		ctx := context.Background()
		segments := []TextSegment{
			{Index: 0, Text: "第一句", Duration: 2.0},
			{Index: 1, Text: "第二句", Duration: 2.5},
		}
		paths, err := provider.SynthesizeBatch(ctx, segments, "default")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(paths) != 2 {
			t.Errorf("expected 2 paths, got %d", len(paths))
		}
	})
}

type MockTTSProvider struct{}

func (m *MockTTSProvider) Synthesize(ctx context.Context, text, voice string) (*AudioResult, error) {
	return &AudioResult{
		Data:       []byte("mock audio data"),
		Duration:   2.0,
		SampleRate: 24000,
	}, nil
}

func (m *MockTTSProvider) SynthesizeBatch(ctx context.Context, segments []TextSegment, voice string) ([]string, error) {
	paths := make([]string, len(segments))
	for _, seg := range segments {
		paths[seg.Index] = "/tmp/mock_" + formatIndex(seg.Index) + ".wav"
	}
	return paths, nil
}

func (m *MockTTSProvider) Name() string {
	return "mock-tts"
}

func TestAiarkTTSProvider(t *testing.T) {
	t.Run("AiarkTTSProvider implements TTSProvider", func(t *testing.T) {
		provider := NewAiarkTTSProvider("http://localhost:8082", "local", "/tmp/tts")
		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.Name() != "aiark-tts" {
			t.Errorf("expected name 'aiark-tts', got '%s'", provider.Name())
		}
	})
}

func TestSilenceGeneration(t *testing.T) {
	t.Run("silence generation for failed TTS", func(t *testing.T) {
		expectedDuration := 2.5
		silenceData := generateSilence(expectedDuration)

		if len(silenceData) == 0 {
			t.Error("expected non-empty silence data")
		}
	})
}

func TestWavHeader(t *testing.T) {
	header := createWavHeader(48000)
	if len(header) != 44 {
		t.Errorf("expected header length 44, got %d", len(header))
	}

	if header[0] != 'R' || header[1] != 'I' || header[2] != 'F' || header[3] != 'F' {
		t.Error("expected RIFF header")
	}
}