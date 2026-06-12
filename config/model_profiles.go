package config

import (
	"fmt"
	"os"
)

func ConfigForModelProfiles(base Config, llmProfile, sttProfile, ttsProfile, ttsVoice string) (Config, error) {
	resolved := base

	if llmProfile != "" {
		profile, ok := base.Models.LLM[llmProfile]
		if !ok {
			return Config{}, fmt.Errorf("unknown llm_profile: %s", llmProfile)
		}
		resolved.Llm.Provider = valueOrDefault(profile.Provider, resolved.Llm.Provider)
		resolved.Llm.BaseURL = valueOrDefault(profile.BaseURL, resolved.Llm.BaseURL)
		resolved.Llm.ApiKey = resolveProfileAPIKey(profile, resolved.Llm.ApiKey)
		resolved.Llm.Model = valueOrDefault(profile.Model, resolved.Llm.Model)
	}

	if sttProfile != "" {
		profile, ok := base.Models.STT[sttProfile]
		if !ok {
			return Config{}, fmt.Errorf("unknown stt_profile: %s", sttProfile)
		}
		resolved.Transcribe.Provider = valueOrDefault(profile.Provider, resolved.Transcribe.Provider)
		if resolved.Transcribe.Provider == "openai" || resolved.Transcribe.Provider == "xai-oauth" {
			resolved.Transcribe.Openai.BaseUrl = valueOrDefault(profile.BaseURL, resolved.Transcribe.Openai.BaseUrl)
			resolved.Transcribe.Openai.ApiKey = resolveProfileAPIKey(profile, resolved.Transcribe.Openai.ApiKey)
			resolved.Transcribe.Openai.Model = valueOrDefault(profile.Model, resolved.Transcribe.Openai.Model)
		}
	}

	if ttsProfile != "" {
		profile, ok := base.Models.TTS[ttsProfile]
		if !ok {
			return Config{}, fmt.Errorf("unknown tts_profile: %s", ttsProfile)
		}
		if ttsVoice != "" && len(profile.Voices) > 0 && !containsString(profile.Voices, ttsVoice) {
			return Config{}, fmt.Errorf("unknown tts_voice for profile %s: %s", ttsProfile, ttsVoice)
		}
		resolved.Tts.Provider = valueOrDefault(profile.Provider, resolved.Tts.Provider)
		if resolved.Tts.Provider == "openai" || resolved.Tts.Provider == "xai-oauth" {
			resolved.Tts.Openai.BaseUrl = valueOrDefault(profile.BaseURL, resolved.Tts.Openai.BaseUrl)
			resolved.Tts.Openai.ApiKey = resolveProfileAPIKey(profile, resolved.Tts.Openai.ApiKey)
			resolved.Tts.Openai.Model = valueOrDefault(profile.Model, resolved.Tts.Openai.Model)
		}
		resolved.Tts.Voices = append([]string(nil), profile.Voices...)
	}

	return resolved, nil
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func resolveProfileAPIKey(profile ModelProfileConfig, fallback string) string {
	if profile.ApiKeyEnv != "" {
		if value := os.Getenv(profile.ApiKeyEnv); value != "" {
			return value
		}
	}
	return valueOrDefault(profile.ApiKey, fallback)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
