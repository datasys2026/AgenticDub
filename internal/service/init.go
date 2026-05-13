package service

import (
	"krillin-ai/config"
	"krillin-ai/internal/providers/llm"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/aliyun"
	"krillin-ai/pkg/fasterwhisper"
	"krillin-ai/pkg/openai"
	"krillin-ai/pkg/whisper"
	"krillin-ai/pkg/whispercpp"
	"krillin-ai/pkg/whisperkit"
	"krillin-ai/pkg/localtts"

	"go.uber.org/zap"
)

type Service struct {
	Transcriber      types.Transcriber
	ChatCompleter    types.ChatCompleter
	TtsClient        types.Ttser
	OssClient        *aliyun.OssClient
	VoiceCloneClient *aliyun.VoiceCloneClient
}

func NewService() *Service {
	var transcriber types.Transcriber
	var chatCompleter types.ChatCompleter
	var ttsClient types.Ttser

	switch config.Conf.Transcribe.Provider {
	case "openai":
		transcriber = whisper.NewClient(config.Conf.Transcribe.Openai.BaseUrl, config.Conf.Transcribe.Openai.ApiKey, config.Conf.Transcribe.Openai.Model, config.Conf.App.Proxy)
	case "fasterwhisper":
		transcriber = fasterwhisper.NewFastwhisperProcessor(config.Conf.Transcribe.Fasterwhisper.Model)
	case "whispercpp":
		transcriber = whispercpp.NewWhispercppProcessor(config.Conf.Transcribe.Whispercpp.Model)
	case "whisperkit":
		transcriber = whisperkit.NewWhisperKitProcessor(config.Conf.Transcribe.Whisperkit.Model)
	case "aliyun":
		cc, err := aliyun.NewAsrClient(config.Conf.Transcribe.Aliyun.Speech.AccessKeyId, config.Conf.Transcribe.Aliyun.Speech.AccessKeySecret, config.Conf.Transcribe.Aliyun.Speech.AppKey, true)
		if err != nil {
			log.GetLogger().Error("创建阿里云语音识别客户端失败： ", zap.Error(err))
			return nil
		}
		transcriber = cc
	}
	log.GetLogger().Info("当前选择的转录源： ", zap.String("transcriber", config.Conf.Transcribe.Provider))

	switch config.Conf.Llm.Provider {
	case "openai":
		provider := llm.NewOpenAIProvider(config.Conf.Llm.BaseURL, config.Conf.Llm.ApiKey, config.Conf.Llm.Model, config.Conf.Llm.ProxyAddr)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	case "aiark":
		provider := llm.NewAiarkLLMProvider(config.Conf.Llm.BaseURL, config.Conf.Llm.ApiKey, config.Conf.Llm.Model)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	default:
		provider := llm.NewOpenAIProvider(config.Conf.Llm.BaseURL, config.Conf.Llm.ApiKey, config.Conf.Llm.Model, config.Conf.Llm.ProxyAddr)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	}
	log.GetLogger().Info("当前选择的LLM： ", zap.String("llm", config.Conf.Llm.Provider))

	switch config.Conf.Tts.Provider {
	case "openai":
		ttsClient = openai.NewClient(config.Conf.Tts.Openai.BaseUrl, config.Conf.Tts.Openai.ApiKey, config.Conf.App.Proxy)
	case "aliyun":
		ttsClient = aliyun.NewTtsClient(config.Conf.Tts.Aliyun.Speech.AccessKeyId, config.Conf.Tts.Aliyun.Speech.AccessKeySecret, config.Conf.Tts.Aliyun.Speech.AppKey)
	case "edge-tts":
		ttsClient = localtts.NewEdgeTtsClient()
	}

	return &Service{
		Transcriber:      transcriber,
		ChatCompleter:    chatCompleter,
		TtsClient:        ttsClient,
		OssClient:        aliyun.NewOssClient(config.Conf.Transcribe.Aliyun.Oss.AccessKeyId, config.Conf.Transcribe.Aliyun.Oss.AccessKeySecret, config.Conf.Transcribe.Aliyun.Oss.Bucket),
		VoiceCloneClient: aliyun.NewVoiceCloneClient(config.Conf.Tts.Aliyun.Speech.AccessKeyId, config.Conf.Tts.Aliyun.Speech.AccessKeySecret, config.Conf.Tts.Aliyun.Speech.AppKey),
	}
}
