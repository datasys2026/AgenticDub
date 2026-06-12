package service

import (
	"fmt"
	"krillin-ai/internal/dto"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/pkg/util"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	slugTokenPattern = regexp.MustCompile(`[A-Za-z0-9]+`)
	safeIDPattern    = regexp.MustCompile(`[^A-Za-z0-9-]+`)
)

func buildTaskID(req dto.StartVideoSubtitleTaskReq, now time.Time) string {
	source := videoSourceName(req.Url)
	sourceID := videoSourceID(req.Url)
	lang := normalizeTargetLanguageCode(req.TargetLang)

	parts := []string{source, sourceID}
	if lang != "" && lang != "none" {
		parts = append(parts, lang)
	}
	parts = append(parts, now.Format("2006-01-02"))

	return strings.Join(nonEmptyNameParts(parts...), "_")
}

func uniqueTaskID(base string) string {
	if base == "" {
		base = "video_" + time.Now().Format("2006-01-02")
	}
	for i := 1; ; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		if _, ok := storage.SubtitleTasks.Load(candidate); ok {
			continue
		}
		if _, err := os.Stat(filepath.Join("./tasks", candidate)); err == nil {
			continue
		}
		return candidate
	}
}

func buildEmbeddedVideoFileName(stepParam *types.SubtitleTaskStepParam, isHorizontal bool, now time.Time) string {
	orientation := "vertical"
	if isHorizontal {
		orientation = "horizontal"
	}
	mode := "subtitled"
	if stepParam.EnableTts {
		mode = "dubbed"
	}

	title := ""
	if stepParam.TaskPtr != nil {
		title = slugifyNamePart(stepParam.TaskPtr.Title)
	}

	parts := []string{
		videoSourceName(stepParam.Link),
		videoSourceID(stepParam.Link),
		title,
		string(stepParam.TargetLanguage),
		orientation,
		mode,
		now.Format("2006-01-02"),
	}
	return strings.Join(nonEmptyNameParts(parts...), "_") + ".mp4"
}

func normalizeTargetLanguageCode(lang string) string {
	switch strings.TrimSpace(lang) {
	case "繁體中文":
		return "zh_tw"
	case "簡體中文":
		return "zh_cn"
	default:
		return strings.ReplaceAll(slugifyNamePart(lang), "-", "_")
	}
}

func videoSourceName(link string) string {
	switch {
	case strings.Contains(link, "youtube.com") || strings.Contains(link, "youtu.be"):
		return "youtube"
	case strings.Contains(link, "bilibili.com"):
		return "bilibili"
	case strings.HasPrefix(link, "local:"):
		return "local"
	default:
		return "video"
	}
}

func videoSourceID(link string) string {
	if strings.Contains(link, "youtube.com") || strings.Contains(link, "youtu.be") {
		if id, err := util.GetYouTubeID(link); err == nil && id != "" {
			return safeIDPart(id)
		}
		if id := extractYouTubeID(link); id != "" {
			return safeIDPart(id)
		}
	}
	if strings.Contains(link, "bilibili.com") {
		if id := util.GetBilibiliVideoId(link); id != "" {
			return safeIDPart(id)
		}
	}
	if strings.HasPrefix(link, "local:") {
		localPath := strings.TrimPrefix(link, "local:")
		return slugifyNamePart(strings.TrimSuffix(filepath.Base(localPath), filepath.Ext(localPath)))
	}
	if parsed, err := url.Parse(link); err == nil {
		if base := strings.Trim(parsed.Path, "/"); base != "" {
			parts := strings.Split(base, "/")
			return slugifyNamePart(parts[len(parts)-1])
		}
		return slugifyNamePart(parsed.Host)
	}
	return slugifyNamePart(link)
}

func nonEmptyNameParts(parts ...string) []string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(part, "_- ")
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}

func safeIDPart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "_", "-")
	value = safeIDPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func slugifyNamePart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	tokens := slugTokenPattern.FindAllString(value, -1)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, "-")
}
