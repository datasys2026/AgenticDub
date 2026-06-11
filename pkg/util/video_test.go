package util

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildMixDubbedAudioArgs(t *testing.T) {
	args := buildMixDubbedAudioArgs("input.mp4", "dubbed.wav", "output.mp4", 0.18, 1.0)
	joined := strings.Join(args, " ")

	expectedParts := []string{
		"-y",
		"-i input.mp4",
		"-i dubbed.wav",
		"volume=0.18",
		"volume=1.00",
		"amix=inputs=2:duration=first:dropout_transition=0",
		"-map 0:v:0",
		"-map [aout]",
		"-c:v copy",
		"-c:a aac",
		"-b:a 192k",
		"output.mp4",
	}

	for _, part := range expectedParts {
		if !strings.Contains(joined, part) {
			t.Fatalf("expected args to contain %q, got %q", part, joined)
		}
	}
	if slices.Contains(args, "-map 1:a:0") {
		t.Fatalf("expected mixed audio output instead of replacing with dubbed track: %v", args)
	}
}
