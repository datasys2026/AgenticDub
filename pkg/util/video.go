package util

import (
	"fmt"
	"krillin-ai/internal/storage"
	"os/exec"
	"strconv"
	"strings"
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
	videoDuration, err := GetAudioDuration(videoFile)
	if err != nil {
		return fmt.Errorf("error probing video duration: %v", err)
	}
	dubbedDuration, err := GetAudioDuration(dubbedAudioFile)
	if err != nil {
		return fmt.Errorf("error probing dubbed audio duration: %v", err)
	}
	width, height, err := GetVideoResolution(videoFile)
	if err != nil {
		return fmt.Errorf("error probing video resolution: %v", err)
	}

	extensionDuration := dubbedDuration - videoDuration
	if extensionDuration < 0.05 {
		extensionDuration = 0
	}

	cmdArgs := buildMixDubbedAudioArgs(videoFile, dubbedAudioFile, outputFile, originalVolume, dubbedVolume, videoDuration, extensionDuration, width, height)
	cmd := exec.Command(storage.FfmpegPath, cmdArgs...)

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("error mixing dubbed audio into video: %v", err)
	}

	return nil
}

func buildMixDubbedAudioArgs(videoFile string, dubbedAudioFile string, outputFile string, originalVolume, dubbedVolume, videoDuration, extensionDuration float64, width, height int) []string {
	if extensionDuration > 0 {
		videoFilter := buildLongDubVideoFilter(videoDuration, extensionDuration, width, height)
		filter := fmt.Sprintf("[0:v]%s[vout];[0:a]volume=%.2f[a0];[1:a]volume=%.2f[a1];[a0][a1]amix=inputs=2:duration=longest:dropout_transition=0[aout]", videoFilter, originalVolume, dubbedVolume)
		return []string{
			"-y",
			"-i", videoFile,
			"-i", dubbedAudioFile,
			"-filter_complex", filter,
			"-map", "[vout]",
			"-map", "[aout]",
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-crf", "18",
			"-pix_fmt", "yuv420p",
			"-c:a", "aac",
			"-b:a", "192k",
			outputFile,
		}
	}

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

func buildLongDubVideoFilter(videoDuration, extensionDuration float64, width, height int) string {
	zoom := fmt.Sprintf("1+0.060*clip((t-%.3f)/%.3f\\,0\\,1)", videoDuration, extensionDuration)
	return fmt.Sprintf("tpad=stop_mode=clone:stop_duration=%.3f,scale=w='trunc(%d*(%s)/2)*2':h='trunc(%d*(%s)/2)*2':eval=frame,crop=%d:%d,fade=t=out:st=%.3f:d=%.3f",
		extensionDuration,
		width,
		zoom,
		height,
		zoom,
		width,
		height,
		videoDuration,
		extensionDuration)
}

func GetVideoResolution(videoFile string) (int, int, error) {
	cmd := exec.Command(storage.FfprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		videoFile,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resolution output: %s", strings.TrimSpace(string(output)))
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(strings.TrimSuffix(parts[1], "x"))
	if err != nil {
		return 0, 0, err
	}
	return width, height, nil
}
