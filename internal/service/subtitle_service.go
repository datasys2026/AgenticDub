package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"krillin-ai/internal/agent/hitl"
	"krillin-ai/internal/dto"
	"krillin-ai/internal/storage"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/util"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

var reviewPollInterval = 5 * time.Second

func (s Service) StartSubtitleTask(req dto.StartVideoSubtitleTaskReq) (*dto.StartVideoSubtitleTaskResData, error) {
	// 校验链接
	if strings.Contains(req.Url, "youtube.com") {
		videoId, _ := util.GetYouTubeID(req.Url)
		if videoId == "" {
			return nil, fmt.Errorf("链接不合法")
		}
	}
	if strings.Contains(req.Url, "bilibili.com") {
		videoId := util.GetBilibiliVideoId(req.Url)
		if videoId == "" {
			return nil, fmt.Errorf("链接不合法")
		}
	}
	// 生成可读任务 ID，便于从 tasks/ 目录直接识别来源与目标语言。
	taskId := uniqueTaskID(buildTaskID(req, time.Now()))
	// 构造任务所需参数
	var resultType types.SubtitleResultType
	// 根据入参选项确定要返回的字幕类型
	if req.TargetLang == "none" {
		resultType = types.SubtitleResultTypeOriginOnly
	} else {
		if req.Bilingual == types.SubtitleTaskBilingualYes {
			if req.TranslationSubtitlePos == types.SubtitleTaskTranslationSubtitlePosTop {
				resultType = types.SubtitleResultTypeBilingualTranslationOnTop
			} else {
				resultType = types.SubtitleResultTypeBilingualTranslationOnBottom
			}
		} else {
			resultType = types.SubtitleResultTypeTargetOnly
		}
	}
	// 文字替换map
	replaceWordsMap := make(map[string]string)
	if len(req.Replace) > 0 {
		for _, replace := range req.Replace {
			beforeAfter := strings.Split(replace, "|")
			if len(beforeAfter) == 2 {
				replaceWordsMap[beforeAfter[0]] = beforeAfter[1]
			} else {
				log.GetLogger().Info("generateAudioSubtitles replace param length err", zap.Any("replace", replace), zap.Any("taskId", taskId))
			}
		}
	}
	var err error
	ctx := context.Background()
	// 创建字幕任务文件夹
	taskBasePath := filepath.Join("./tasks", taskId)
	if _, err = os.Stat(taskBasePath); os.IsNotExist(err) {
		// 不存在则创建
		err = os.MkdirAll(filepath.Join(taskBasePath, "output"), os.ModePerm)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask MkdirAll err", zap.Any("req", req), zap.Error(err))
		}
	}

	// 创建任务
	taskPtr := &types.SubtitleTask{
		TaskId:   taskId,
		VideoSrc: req.Url,
		Status:   types.SubtitleTaskStatusProcessing,
	}
	storage.SubtitleTasks.Store(taskId, taskPtr)

	// 处理声音克隆源
	var voiceCloneAudioUrl string
	if req.TtsVoiceCloneSrcFileUrl != "" {
		localFileUrl := strings.TrimPrefix(req.TtsVoiceCloneSrcFileUrl, "local:")
		fileKey := util.GenerateRandStringWithUpperLowerNum(5) + filepath.Ext(localFileUrl) // 防止url encode的问题，这里统一处理
		err = s.OssClient.UploadFile(context.Background(), fileKey, localFileUrl, s.OssClient.Bucket)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask UploadFile err", zap.Any("req", req), zap.Error(err))
			return nil, errors.New("上传声音克隆源失败")
		}
		voiceCloneAudioUrl = fmt.Sprintf("https://%s.oss-cn-shanghai.aliyuncs.com/%s", s.OssClient.Bucket, fileKey)
		log.GetLogger().Info("StartVideoSubtitleTask 上传声音克隆源成功", zap.Any("oss url", voiceCloneAudioUrl))
	}

	stepParam := types.SubtitleTaskStepParam{
		TaskId:             taskId,
		TaskPtr:            taskPtr,
		TaskBasePath:       taskBasePath,
		Link:               req.Url,
		SubtitleResultType: resultType,
		EnableModalFilter:  req.ModalFilter == types.SubtitleTaskModalFilterYes,
		EnableTts:          req.Tts == types.SubtitleTaskTtsYes,
		TtsVoiceCode:       req.TtsVoiceCode,
		VoiceCloneAudioUrl: voiceCloneAudioUrl,
		ReplaceWordsMap:    replaceWordsMap,
		OriginLanguage:     types.StandardLanguageCode(req.OriginLanguage),
		UserUILanguage:     types.StandardLanguageCode(req.Language),
	}
	stepParam.TargetLanguage = types.StandardLanguageCode(normalizeTargetLanguageCode(req.TargetLang))
	embedSubtitleType := req.EmbedSubtitleVideoType
	if embedSubtitleType == "" {
		embedSubtitleType = "horizontal"
	}
	stepParam.EmbedSubtitleVideoType = embedSubtitleType
	stepParam.VerticalVideoMajorTitle = req.VerticalMajorTitle
	stepParam.VerticalVideoMinorTitle = req.VerticalMinorTitle
	stepParam.MaxWordOneLine = 12 // 默认值
	if req.OriginLanguageWordOneLine != 0 {
		stepParam.MaxWordOneLine = req.OriginLanguageWordOneLine
	}

	log.GetLogger().Info("current task info", zap.String("taskId", taskId), zap.Any("param", stepParam))

	go func() {
		defer func() {
			if r := recover(); r != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]
				log.GetLogger().Error("autoVideoSubtitle panic", zap.Any("panic:", r), zap.Any("stack:", buf))
				stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			}
		}()
		// 新版流程：链接->本地音频文件->视频信息获取（若有）->本地字幕文件->语言合成->视频合成->字幕文件链接生成
		log.GetLogger().Info("video subtitle start task", zap.String("taskId", taskId))
		err = s.linkToFile(ctx, &stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask linkToFile err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}
		s.populateTaskTitle(ctx, &stepParam)
		// 暂时不加视频信息
		//err = s.getVideoInfo(ctx, &stepParam)
		//if err != nil {
		//	log.GetLogger().Error("StartVideoSubtitleTask getVideoInfo err", zap.Any("req", req), zap.Error(err))
		//	stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
		//	stepParam.TaskPtr.FailReason = "get video info error"
		//	return
		//}
		err = s.audioToSubtitle(ctx, &stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask audioToSubtitle err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}

		// HITL: Generate review.txt and wait for review.
		// bilingual_srt.srt contains line1=original, line2=target unless translation-on-top is requested.
		bilingualSrtPath := filepath.Join(stepParam.TaskBasePath, types.SubtitleTaskBilingualSrtFileName)
		hitlSvc := s.getHITLService()
		targetOnTop := stepParam.SubtitleResultType == types.SubtitleResultTypeBilingualTranslationOnTop
		doc, err := hitlSvc.CreateReviewFromBilingual(stepParam.TaskId, bilingualSrtPath, stepParam.TaskPtr.Title, string(stepParam.TargetLanguage), targetOnTop)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask CreateReview err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}
		doc = s.auditAndSuggestReview(stepParam.TaskBasePath, doc, stepParam.TargetLanguage)

		reviewPath := filepath.Join(stepParam.TaskBasePath, "review.txt")
		err = hitlSvc.SaveReview(doc, reviewPath)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask SaveReview err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}

		// Set status to pending review
		stepParam.TaskPtr.Status = types.SubtitleTaskStatusPendingReview
		stepParam.TaskPtr.ProcessPct = 90
		log.GetLogger().Info("video subtitle task pending review", zap.String("taskId", taskId), zap.String("reviewPath", reviewPath))

		// Wait for review (blocking loop)
		if err := s.waitForReview(&stepParam, reviewPath, ctx); err != nil {
			log.GetLogger().Warn("video subtitle task stopped after review", zap.String("taskId", taskId), zap.Error(err))
			if stepParam.TaskPtr.Status != types.SubtitleTaskStatusFailed {
				stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
				stepParam.TaskPtr.FailReason = err.Error()
			}
			return
		}

		// Review approved, continue with TTS
		log.GetLogger().Info("video subtitle continue after review", zap.String("taskId", taskId))
		err = s.srtFileToSpeech(ctx, &stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask srtFileToSpeech err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}
		err = s.embedSubtitles(ctx, &stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask embedSubtitles err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}
		err = s.uploadSubtitles(ctx, &stepParam)
		if err != nil {
			log.GetLogger().Error("StartVideoSubtitleTask uploadSubtitles err", zap.Any("req", req), zap.Error(err))
			stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
			stepParam.TaskPtr.FailReason = err.Error()
			return
		}

		log.GetLogger().Info("video subtitle task end", zap.String("taskId", taskId))
	}()

	return &dto.StartVideoSubtitleTaskResData{
		TaskId: taskId,
	}, nil
}

func (s Service) GetTaskStatus(req dto.GetVideoSubtitleTaskReq) (*dto.GetVideoSubtitleTaskResData, error) {
	task, ok := storage.SubtitleTasks.Load(req.TaskId)
	if !ok || task == nil {
		return nil, errors.New("任务不存在")
	}
	taskPtr := task.(*types.SubtitleTask)
	if taskPtr.Status == types.SubtitleTaskStatusFailed {
		return nil, fmt.Errorf("任务失败，原因：%s", taskPtr.FailReason)
	}
	return &dto.GetVideoSubtitleTaskResData{
		TaskId:         taskPtr.TaskId,
		ProcessPercent: taskPtr.ProcessPct,
		VideoInfo: &dto.VideoInfo{
			Title:                 taskPtr.Title,
			Description:           taskPtr.Description,
			TranslatedTitle:       taskPtr.TranslatedTitle,
			TranslatedDescription: taskPtr.TranslatedDescription,
		},
		SubtitleInfo: lo.Map(taskPtr.SubtitleInfos, func(item types.SubtitleInfo, _ int) *dto.SubtitleInfo {
			return &dto.SubtitleInfo{
				Name:        item.Name,
				DownloadUrl: item.DownloadUrl,
			}
		}),
		TargetLanguage:    taskPtr.TargetLanguage,
		SpeechDownloadUrl: taskPtr.SpeechDownloadUrl,
		VideoDownloadUrl:  taskPtr.VideoDownloadUrl,
	}, nil
}

func (s Service) getHITLService() hitl.ReviewService {
	baseDir, _ := os.Getwd()
	return hitl.NewReviewService(
		hitl.TxtParser{},
		hitl.SRTMerger{},
		filepath.Join(baseDir, "tasks"),
	)
}

func (s Service) waitForReview(stepParam *types.SubtitleTaskStepParam, reviewPath string, ctx context.Context) error {
	ticker := time.NewTicker(reviewPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			task, ok := storage.SubtitleTasks.Load(stepParam.TaskId)
			if !ok {
				return fmt.Errorf("task %s not found while waiting for review", stepParam.TaskId)
			}
			taskPtr := task.(*types.SubtitleTask)

			// Check approval/rejection marker first. The HTTP handler may already
			// move the task back to processing, but the edited review still must be applied.
			statusPath := filepath.Join(stepParam.TaskBasePath, "status.json")
			if data, err := os.ReadFile(statusPath); err == nil {
				var status hitl.TaskStatus
				if err := json.Unmarshal(data, &status); err == nil {
					if status.Status == hitl.StatusApproved {
						if err := s.applyApprovedReview(stepParam, reviewPath); err != nil {
							log.GetLogger().Error("waitForReview applyApprovedReview err", zap.Error(err))
							return err
						}
						return nil
					}
					if status.Status == hitl.StatusRejected {
						stepParam.TaskPtr.Status = types.SubtitleTaskStatusFailed
						stepParam.TaskPtr.FailReason = status.RejectReason
						return fmt.Errorf("review rejected: %s", status.RejectReason)
					}
				}
			}

			// Check if status changed from pending_review without a marker.
			if taskPtr.Status == types.SubtitleTaskStatusProcessing && taskPtr.ProcessPct > 90 {
				return nil
			}
		}
	}
}

func (s Service) applyApprovedReview(stepParam *types.SubtitleTaskStepParam, reviewPath string) error {
	targetPath := filepath.Join(stepParam.TaskBasePath, types.SubtitleTaskTargetLanguageSrtFileName)
	translatedPath := filepath.Join(stepParam.TaskBasePath, "translated.srt")
	if _, err := os.Stat(translatedPath); os.IsNotExist(err) {
		if err := copyFile(targetPath, translatedPath); err != nil {
			return err
		}
	}

	hitlSvc := s.getHITLService()
	finalSRTPath, err := hitlSvc.Approve(stepParam.TaskId, reviewPath)
	if err != nil {
		return err
	}

	if stepParam.SubtitleResultType == types.SubtitleResultTypeTargetOnly {
		if err := copyFile(finalSRTPath, targetPath); err != nil {
			return err
		}
		stepParam.TtsSourceFilePath = targetPath
	} else {
		stepParam.TtsSourceFilePath = finalSRTPath
	}

	stepParam.TaskPtr.Status = types.SubtitleTaskStatusProcessing
	stepParam.TaskPtr.ProcessPct = 91
	return nil
}

func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}
