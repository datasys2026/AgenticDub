package util

import (
	"fmt"
	"krillin-ai/internal/storage"
	"os/exec"
)

const (
	DefaultOriginalAudioVolume = 0.18
	DefaultDubbedAudioVolume   = 1.00
)

func ReplaceAudioInVideo(videoFile string, audioFile string, outputFile string) error {
	cmd := exec.Command(storage.FfmpegPath, "-i", videoFile, "-i", audioFile, "-c:v", "copy", "-map", "0:v:0", "-map", "1:a:0", outputFile)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error replacing audio in video: %v", err)
	}

	return nil
}

func MixDubbedAudioInVideo(videoFile string, dubbedAudioFile string, outputFile string, originalVolume, dubbedVolume float64) error {
	cmdArgs := buildMixDubbedAudioArgs(videoFile, dubbedAudioFile, outputFile, originalVolume, dubbedVolume)
	cmd := exec.Command(storage.FfmpegPath, cmdArgs...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error mixing dubbed audio into video: %v", err)
	}

	return nil
}

func buildMixDubbedAudioArgs(videoFile string, dubbedAudioFile string, outputFile string, originalVolume, dubbedVolume float64) []string {
	filter := fmt.Sprintf("[0:a]volume=%.2f[a0];[1:a]volume=%.2f[a1];[a0][a1]amix=inputs=2:duration=first:dropout_transition=0[aout]", originalVolume, dubbedVolume)
	return []string{
		"-y",
		"-i", videoFile,
		"-i", dubbedAudioFile,
		"-filter_complex", filter,
		"-map", "0:v:0",
		"-map", "[aout]",
		"-c:v", "copy",
		"-c:a", "aac",
		"-b:a", "192k",
		outputFile,
	}
}
