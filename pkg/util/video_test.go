package util

import (
	"slices"
	"strings"
	"testing"
)

func TestBuildMixDubbedAudioArgs(t *testing.T) {
	args := buildMixDubbedAudioArgs("input.mp4", "dubbed.wav", "output.mp4", 0.18, 1.0, 501.435, 0, 1920, 1080)
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

func TestBuildMixDubbedAudioArgsExtendsVideoForLongDub(t *testing.T) {
	args := buildMixDubbedAudioArgs("input.mp4", "dubbed.wav", "output.mp4", 0.18, 1.0, 501.435, 22.245, 1920, 1080)
	joined := strings.Join(args, " ")

	expectedParts := []string{
		"tpad=stop_mode=clone:stop_duration=22.245",
		"scale=w='trunc(1920*(1+0.060*clip((t-501.435)/22.245\\,0\\,1))/2)*2'",
		"h='trunc(1080*(1+0.060*clip((t-501.435)/22.245\\,0\\,1))/2)*2'",
		"crop=1920:1080",
		"fade=t=out:st=501.435:d=22.245",
		"amix=inputs=2:duration=longest:dropout_transition=0",
		"-map [vout]",
		"-map [aout]",
		"-c:v libx264",
		"-preset veryfast",
		"-crf 18",
		"-pix_fmt yuv420p",
		"-c:a aac",
	}

	for _, part := range expectedParts {
		if !strings.Contains(joined, part) {
			t.Fatalf("expected args to contain %q, got %q", part, joined)
		}
	}
	if strings.Contains(joined, "-c:v copy") {
		t.Fatalf("extended video cannot copy video stream: %q", joined)
	}
}
