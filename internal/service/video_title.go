package service

import (
	"context"
	"krillin-ai/config"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

func (s Service) populateTaskTitle(ctx context.Context, stepParam *types.SubtitleTaskStepParam) {
	if stepParam == nil || stepParam.TaskPtr == nil || strings.TrimSpace(stepParam.TaskPtr.Title) != "" {
		return
	}

	title, err := taskTitleFromSource(ctx, stepParam.Link)
	if err != nil {
		log.GetLogger().Warn("populateTaskTitle skipped", zap.String("taskId", stepParam.TaskId), zap.Error(err))
		return
	}
	title = strings.TrimSpace(title)
	if title != "" {
		stepParam.TaskPtr.Title = title
	}
}

func taskTitleFromSource(ctx context.Context, link string) (string, error) {
	if strings.HasPrefix(link, "local:") {
		localPath := strings.TrimPrefix(link, "local:")
		base := filepath.Base(localPath)
		return strings.TrimSuffix(base, filepath.Ext(base)), nil
	}
	if strings.Contains(link, "youtube.com") || strings.Contains(link, "youtu.be") || strings.Contains(link, "bilibili.com") {
		return fetchRemoteVideoTitle(ctx, link)
	}
	return "", nil
}

func fetchRemoteVideoTitle(ctx context.Context, link string) (string, error) {
	args := []string{"--skip-download", "--encoding", "utf-8", "--get-title", link}
	if strings.Contains(link, "youtube.com") || strings.Contains(link, "youtu.be") {
		args = append(args, "--cookies", "./cookies.txt")
	}
	if config.Conf.App.Proxy != "" {
		args = append(args, "--proxy", config.Conf.App.Proxy)
	}
	if storage.FfmpegPath != "ffmpeg" {
		args = append(args, "--ffmpeg-location", storage.FfmpegPath)
	}

	output, err := exec.CommandContext(ctx, storage.YtdlpPath, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
