package config

import (
	"errors"
	"fmt"
	"krillin-ai/log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
	"go.uber.org/zap"
)

var ConfigBackup Config // 用于在开始任务之前，检测配置是否更新，更新后要重启服务端

type App struct {
	SegmentDuration       int      `toml:"segment_duration"`
	TranscribeParallelNum int      `toml:"transcribe_parallel_num"`
	TranslateParallelNum  int      `toml:"translate_parallel_num"`
	TranscribeMaxAttempts int      `toml:"transcribe_max_attempts"`
	TranslateMaxAttempts  int      `toml:"translate_max_attempts"`
	MaxSentenceLength     int      `toml:"max_sentence_length"`
	Proxy                 string   `toml:"proxy"`
	ParsedProxy           *url.URL `toml:"-"`
}

type Server struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type OpenaiCompatibleConfig struct {
	BaseUrl string `toml:"base_url"`
	ApiKey  string `toml:"api_key"`
	Model   string `toml:"model"`
}

type LocalModelConfig struct {
	Model string `toml:"model"`
}

type AliyunSpeechConfig struct {
	AccessKeyId     string `toml:"access_key_id"`
	AccessKeySecret string `toml:"access_key_secret"`
	AppKey          string `toml:"app_key"`
}

type AliyunOssConfig struct {
	AccessKeyId     string `toml:"access_key_id"`
	AccessKeySecret string `toml:"access_key_secret"`
	Bucket          string `toml:"bucket"`
}

type AliyunTranscribeConfig struct {
	Oss    AliyunOssConfig    `toml:"oss"`
	Speech AliyunSpeechConfig `toml:"speech"`
}

type Transcribe struct {
	Provider              string                 `toml:"provider"`
	EnableGpuAcceleration bool                   `toml:"enable_gpu_acceleration"`
	Openai                OpenaiCompatibleConfig `toml:"openai"`
	Fasterwhisper         LocalModelConfig       `toml:"fasterwhisper"`
	Whisperkit            LocalModelConfig       `toml:"whisperkit"`
	Whispercpp            LocalModelConfig       `toml:"whispercpp"`
	Aliyun                AliyunTranscribeConfig `toml:"aliyun"`
}

type AliyunTtsConfig struct {
	Oss    AliyunOssConfig    `toml:"oss"`
	Speech AliyunSpeechConfig `toml:"speech"`
}

type Tts struct {
	Provider string                 `toml:"provider"`
	Openai   OpenaiCompatibleConfig `toml:"openai"`
	Aliyun   AliyunTtsConfig        `toml:"aliyun"`
	Voices   []string               `toml:"voices"`
}

type OpenAiWhisper struct {
	BaseUrl string `toml:"base_url"`
	ApiKey  string `toml:"api_key"`
}

type Config struct {
	App        App                 `toml:"app"`
	Server     Server              `toml:"server"`
	Llm        LLMConfig           `toml:"llm"`
	Transcribe Transcribe          `toml:"transcribe"`
	Tts        Tts                 `toml:"tts"`
	Mcp        McpConfig           `toml:"mcp"`
	XAI        XAIConfig           `toml:"xai_oauth"`
	Models     ModelRegistryConfig `toml:"models"`
}

type McpConfig struct {
	ServerURL string `toml:"server_url"`
}

type XAIConfig struct {
	BaseURL   string `toml:"base_url"`
	TokenPath string `toml:"token_path"`
}

type LLMConfig struct {
	Provider  string `toml:"provider"`
	Model     string `toml:"model"`
	BaseURL   string `toml:"base_url"`
	ApiKey    string `toml:"api_key"`
	ProxyAddr string `toml:"proxy_addr"`
}

type ModelProfileConfig struct {
	Provider  string   `toml:"provider"`
	BaseURL   string   `toml:"base_url"`
	ApiKey    string   `toml:"api_key"`
	ApiKeyEnv string   `toml:"api_key_env"`
	Model     string   `toml:"model"`
	Voices    []string `toml:"voices"`
}

type ModelRegistryConfig struct {
	LLM map[string]ModelProfileConfig `toml:"llm"`
	STT map[string]ModelProfileConfig `toml:"stt"`
	TTS map[string]ModelProfileConfig `toml:"tts"`
}

var Conf = Config{
	App: App{
		SegmentDuration:       5,
		TranslateParallelNum:  3,
		TranscribeParallelNum: 1,
		TranscribeMaxAttempts: 3,
		TranslateMaxAttempts:  3,
		MaxSentenceLength:     70,
	},
	Server: Server{
		Host: "127.0.0.1",
		Port: 8888,
	},
	Llm: LLMConfig{
		Provider: "aiark",
		Model:    "aiark/gemma4-e2b",
		BaseURL:  "https://aiark.com.tw/v1",
	},
	XAI: XAIConfig{
		BaseURL:   "https://api.x.ai/v1",
		TokenPath: "~/.agenticdub/auth/xai.json",
	},
	Transcribe: Transcribe{
		Provider:              "openai",
		EnableGpuAcceleration: false, // 默认不开启GPU加速
		Openai: OpenaiCompatibleConfig{
			Model: "whisper-1",
		},
		Fasterwhisper: LocalModelConfig{
			Model: "large-v2",
		},
		Whisperkit: LocalModelConfig{
			Model: "large-v2",
		},
		Whispercpp: LocalModelConfig{
			Model: "large-v2",
		},
	},
	Tts: Tts{
		Provider: "openai",
		Openai: OpenaiCompatibleConfig{
			Model: "gpt-4o-mini-tts",
		},
	},
	Models: ModelRegistryConfig{
		LLM: map[string]ModelProfileConfig{
			"fast": {
				Provider:  "aiark",
				BaseURL:   "https://aiark.com.tw/v1",
				ApiKeyEnv: "AIARK_LLM_API_KEY",
				Model:     "aiark/qwen36-35b",
			},
			"quality": {
				Provider:  "aiark",
				BaseURL:   "https://aiark.com.tw/v1",
				ApiKeyEnv: "AIARK_LLM_API_KEY",
				Model:     "aiark/gemma4-31b-qat",
			},
			"external": {
				Provider:  "aiark",
				BaseURL:   "https://aiark.com.tw/v1",
				ApiKeyEnv: "AIARK_LLM_API_KEY",
				Model:     "aiark/gemma4-26b-qat",
			},
			"light": {
				Provider:  "aiark",
				BaseURL:   "https://aiark.com.tw/v1",
				ApiKeyEnv: "AIARK_LLM_API_KEY",
				Model:     "aiark/gemma4-e4b",
			},
			"grok": {
				Provider: "xai-oauth",
				BaseURL:  "https://api.x.ai/v1",
				Model:    "grok-4.20-0309-non-reasoning",
			},
		},
		STT: map[string]ModelProfileConfig{
			"default": {
				Provider:  "openai",
				BaseURL:   "https://aiark.com.tw/v1",
				ApiKeyEnv: "AIARK_STT_API_KEY",
				Model:     "aiark/faster-whisper-large-v3-fp16",
			},
			"xai": {
				Provider: "xai-oauth",
				BaseURL:  "https://api.x.ai/v1",
				Model:    "xai-stt",
			},
		},
		TTS: map[string]ModelProfileConfig{
			"default": {
				Provider:  "openai",
				BaseURL:   "https://aiark.com.tw/tts/v1",
				ApiKeyEnv: "AIARK_TTS_API_KEY",
				Model:     "aiark/qwen3-tts-0.6b-customvoice",
				Voices: []string{
					"Vivian",
					"Serena",
					"Uncle_Fu",
					"Dylan",
					"Eric",
					"Ryan",
					"Aiden",
					"Ono_Anna",
					"Sohee",
				},
			},
			"xai": {
				Provider: "xai-oauth",
				BaseURL:  "https://api.x.ai/v1",
				Model:    "xai-tts",
				Voices: []string{
					"eve",
					"ara",
					"rex",
					"sal",
					"leo",
				},
			},
		},
	},
}

// 检查必要的配置是否完整
func validateConfig() error {
	// 检查LLM服务提供商
	switch Conf.Llm.Provider {
	case "openai", "aiark", "ollama", "xai-oauth":
	default:
		return fmt.Errorf("不支持的LLM提供商: %s（可选：openai, aiark, ollama, xai-oauth）", Conf.Llm.Provider)
	}

	// 检查转写服务提供商配置
	switch Conf.Transcribe.Provider {
	case "openai":
		if Conf.Transcribe.Openai.ApiKey == "" {
			return errors.New("使用OpenAI转录服务需要配置 OpenAI API Key")
		}
	case "xai-oauth":
	case "fasterwhisper":
		if Conf.Transcribe.Fasterwhisper.Model != "tiny" && Conf.Transcribe.Fasterwhisper.Model != "medium" && Conf.Transcribe.Fasterwhisper.Model != "large-v2" {
			return errors.New("检测到开启了fasterwhisper，但模型选型配置不正确，请检查配置")
		}
	case "whisperkit":
		if runtime.GOOS != "darwin" {
			log.GetLogger().Error("whisperkit只支持macos", zap.String("当前系统", runtime.GOOS))
			return fmt.Errorf("whisperkit只支持macos")
		}
		if Conf.Transcribe.Whisperkit.Model != "large-v2" {
			return errors.New("检测到开启了whisperkit，但模型选型配置不正确，请检查配置")
		}
	case "whispercpp":
		if runtime.GOOS != "windows" {
			return errors.New("whispercpp只支持windows")
		}
		if Conf.Transcribe.Whispercpp.Model != "large-v2" {
			return errors.New("检测到开启了whispercpp，但模型选型配置不正确，请检查配置")
		}
	case "aliyun":
		if Conf.Transcribe.Aliyun.Speech.AccessKeyId == "" || Conf.Transcribe.Aliyun.Speech.AccessKeySecret == "" || Conf.Transcribe.Aliyun.Speech.AppKey == "" {
			return errors.New("使用阿里云语音服务需要配置相关密钥")
		}
	default:
		return errors.New("不支持的转录提供商")
	}

	return nil
}

func LoadConfig() bool {
	var err error
	configPath := "./config/config.toml"
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		log.GetLogger().Info("未找到配置文件")
		return false
	} else {
		log.GetLogger().Info("已找到配置文件，从配置文件中加载配置")
		if _, err = toml.DecodeFile(configPath, &Conf); err != nil {
			log.GetLogger().Error("加载配置文件失败", zap.Error(err))
			return false
		}
		return true
	}
}

// 验证配置
func CheckConfig() error {
	var err error
	// 解析代理地址
	Conf.App.ParsedProxy, err = url.Parse(Conf.App.Proxy)
	if err != nil {
		return err
	}
	return validateConfig()
}

// SaveConfig 保存配置到文件
func SaveConfig() error {
	configPath := filepath.Join("config", "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(configPath), os.ModePerm)
		if err != nil {
			return err
		}
	}

	data, err := toml.Marshal(Conf)
	if err != nil {
		return err
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
