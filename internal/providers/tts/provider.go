package tts

import (
	"context"
	"errors"
	"fmt"
	"os"
)

var (
	ErrSynthesizeFailed = errors.New("TTS synthesis failed")
	ErrEmptyText        = errors.New("text is empty")
)

type AudioResult struct {
	Data       []byte
	Duration   float64
	SampleRate int
}

type TTSProvider interface {
	Synthesize(ctx context.Context, text, voice string) (*AudioResult, error)
	SynthesizeBatch(ctx context.Context, segments []TextSegment, voice string) ([]string, error)
	Name() string
}

type TextSegment struct {
	Index     int
	Text      string
	StartTime float64
	EndTime   float64
	Duration  float64
}

type AiarkTTSProvider struct {
	baseURL   string
	apiKey    string
	model     string
	outputDir string
}

func NewAiarkTTSProvider(baseURL, apiKey, outputDir string) *AiarkTTSProvider {
	return &AiarkTTSProvider{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     "qwen3-tts",
		outputDir: outputDir,
	}
}

func (p *AiarkTTSProvider) Name() string {
	return "aiark-tts"
}

func (p *AiarkTTSProvider) Synthesize(ctx context.Context, text, voice string) (*AudioResult, error) {
	if text == "" {
		return nil, ErrEmptyText
	}

	return &AudioResult{
		Data:       []byte{},
		SampleRate: 24000,
	}, nil
}

func (p *AiarkTTSProvider) SynthesizeBatch(ctx context.Context, segments []TextSegment, voice string) ([]string, error) {
	outPaths := make([]string, len(segments))

	for _, seg := range segments {
		result, err := p.Synthesize(ctx, seg.Text, voice)
		if err != nil {
			outPaths[seg.Index] = ""
			continue
		}

		filename := p.outputDir + "/" + formatIndex(seg.Index) + ".wav"
		err = os.WriteFile(filename, result.Data, 0644)
		if err != nil {
			outPaths[seg.Index] = ""
			continue
		}
		outPaths[seg.Index] = filename
	}

	return outPaths, nil
}

func formatIndex(index int) string {
	return fmt.Sprintf("%04d", index)
}

func generateSilence(duration float64) []byte {
	sampleRate := 24000
	numSamples := int(duration * float64(sampleRate))
	bytesPerSample := 2

	header := createWavHeader(numSamples * bytesPerSample)
	silence := make([]byte, numSamples*bytesPerSample)

	result := append(header, silence...)
	return result
}

func createWavHeader(dataSize int) []byte {
	header := make([]byte, 44)
	sampleRate := 24000
	channels := 1
	bitsPerSample := 16

	header[0] = 'R'
	header[1] = 'I'
	header[2] = 'F'
	header[3] = 'F'
	writeUint32LE(header[4:], uint32(36+dataSize))
	header[8] = 'W'
	header[9] = 'A'
	header[10] = 'V'
	header[11] = 'E'
	header[12] = 'f'
	header[13] = 'm'
	header[14] = 't'
	header[15] = ' '
	writeUint32LE(header[16:], 16)
	writeUint16LE(header[20:], 1)
	writeUint16LE(header[22:], uint16(channels))
	writeUint32LE(header[24:], uint32(sampleRate))
	writeUint32LE(header[28:], uint32(sampleRate*channels*bitsPerSample/8))
	writeUint16LE(header[32:], uint16(channels*bitsPerSample/8))
	writeUint16LE(header[34:], uint16(bitsPerSample))
	header[36] = 'd'
	header[37] = 'a'
	header[38] = 't'
	header[39] = 'a'
	writeUint32LE(header[40:], uint32(dataSize))

	return header
}

func writeUint32LE(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func writeUint16LE(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}