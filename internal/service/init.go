package service

import (
	"krillin-ai/config"
	"krillin-ai/internal/providers/llm"
	"krillin-ai/internal/types"
	"krillin-ai/log"
	"krillin-ai/pkg/aliyun"
	"krillin-ai/pkg/fasterwhisper"
	"krillin-ai/pkg/localtts"
	"krillin-ai/pkg/openai"
	"krillin-ai/pkg/whisper"
	"krillin-ai/pkg/whispercpp"
	"krillin-ai/pkg/whisperkit"

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
	return NewServiceWithConfig(config.Conf)
}

func NewServiceWithConfig(conf config.Config) *Service {
	var transcriber types.Transcriber
	var chatCompleter types.ChatCompleter
	var ttsClient types.Ttser

	switch conf.Transcribe.Provider {
	case "openai":
		transcriber = whisper.NewClient(conf.Transcribe.Openai.BaseUrl, conf.Transcribe.Openai.ApiKey, conf.Transcribe.Openai.Model, conf.App.Proxy)
	case "fasterwhisper":
		transcriber = fasterwhisper.NewFastwhisperProcessor(conf.Transcribe.Fasterwhisper.Model)
	case "whispercpp":
		transcriber = whispercpp.NewWhispercppProcessor(conf.Transcribe.Whispercpp.Model)
	case "whisperkit":
		transcriber = whisperkit.NewWhisperKitProcessor(conf.Transcribe.Whisperkit.Model)
	case "aliyun":
		cc, err := aliyun.NewAsrClient(conf.Transcribe.Aliyun.Speech.AccessKeyId, conf.Transcribe.Aliyun.Speech.AccessKeySecret, conf.Transcribe.Aliyun.Speech.AppKey, true)
		if err != nil {
			log.GetLogger().Error("创建阿里云语音识别客户端失败： ", zap.Error(err))
			return nil
		}
		transcriber = cc
	}
	log.GetLogger().Info("当前选择的转录源： ", zap.String("transcriber", conf.Transcribe.Provider))

	switch conf.Llm.Provider {
	case "openai":
		provider := llm.NewOpenAIProvider(conf.Llm.BaseURL, conf.Llm.ApiKey, conf.Llm.Model, conf.Llm.ProxyAddr)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	case "aiark":
		provider := llm.NewAiarkLLMProvider(conf.Llm.BaseURL, conf.Llm.ApiKey, conf.Llm.Model)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	case "xai-oauth":
		baseURL := conf.Llm.BaseURL
		if baseURL == "" {
			baseURL = conf.XAI.BaseURL
		}
		provider := llm.NewXAIOAuthProviderFromTokenFile(baseURL, conf.Llm.Model, conf.XAI.TokenPath)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	default:
		provider := llm.NewOpenAIProvider(conf.Llm.BaseURL, conf.Llm.ApiKey, conf.Llm.Model, conf.Llm.ProxyAddr)
		chatCompleter = llm.NewChatCompleterAdapter(provider)
	}
	log.GetLogger().Info("当前选择的LLM： ", zap.String("llm", conf.Llm.Provider))

	switch conf.Tts.Provider {
	case "openai":
		ttsClient = openai.NewClient(conf.Tts.Openai.BaseUrl, conf.Tts.Openai.ApiKey, conf.Tts.Openai.Model, conf.App.Proxy)
	case "aliyun":
		ttsClient = aliyun.NewTtsClient(conf.Tts.Aliyun.Speech.AccessKeyId, conf.Tts.Aliyun.Speech.AccessKeySecret, conf.Tts.Aliyun.Speech.AppKey)
	case "edge-tts":
		ttsClient = localtts.NewEdgeTtsClient()
	}

	return &Service{
		Transcriber:      transcriber,
		ChatCompleter:    chatCompleter,
		TtsClient:        ttsClient,
		OssClient:        aliyun.NewOssClient(conf.Transcribe.Aliyun.Oss.AccessKeyId, conf.Transcribe.Aliyun.Oss.AccessKeySecret, conf.Transcribe.Aliyun.Oss.Bucket),
		VoiceCloneClient: aliyun.NewVoiceCloneClient(conf.Tts.Aliyun.Speech.AccessKeyId, conf.Tts.Aliyun.Speech.AccessKeySecret, conf.Tts.Aliyun.Speech.AppKey),
	}
}
