package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"krillin-ai/config"
)

func TestListModelProfilesIncludesXAIProfiles(t *testing.T) {
	originalConf := config.Conf
	t.Cleanup(func() {
		config.Conf = originalConf
	})
	config.Conf.Models.LLM = map[string]config.ModelProfileConfig{
		"grok": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "grok-4.20-0309-non-reasoning",
		},
	}
	config.Conf.Models.STT = map[string]config.ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "xai-stt",
		},
	}
	config.Conf.Models.TTS = map[string]config.ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "xai-tts",
			Voices:   []string{"eve", "ara", "rex", "sal", "leo"},
		},
	}

	_, output, err := ListModelProfiles(context.Background(), nil, ListModelProfilesInput{})
	if err != nil {
		t.Fatalf("ListModelProfiles failed: %v", err)
	}

	grok, ok := output.LLM["grok"]
	if !ok {
		t.Fatal("expected grok LLM profile")
	}
	if grok.Provider != "xai-oauth" {
		t.Fatalf("expected xai-oauth provider, got %q", grok.Provider)
	}
	if grok.Model != "grok-4.20-0309-non-reasoning" {
		t.Fatalf("expected grok-4.20-0309-non-reasoning model, got %q", grok.Model)
	}
	if output.STT["xai"].Model != "xai-stt" {
		t.Fatalf("expected xai-stt STT profile, got %#v", output.STT["xai"])
	}
	if got := output.TTS["xai"].Voices; len(got) != 5 || got[0] != "eve" || got[4] != "leo" {
		t.Fatalf("expected xAI TTS voices, got %#v", got)
	}
}

func TestTranslateVideoLLMProfileSchemaMentionsGrok(t *testing.T) {
	field, ok := reflect.TypeOf(TranslateVideoInput{}).FieldByName("LLMProfile")
	if !ok {
		t.Fatal("expected LLMProfile field")
	}
	if schema := field.Tag.Get("jsonschema"); !strings.Contains(schema, "grok") {
		t.Fatalf("expected LLMProfile schema to mention grok, got %q", schema)
	}
}

func TestTranslateVideoSTTAndTTSProfileSchemaMentionsXAI(t *testing.T) {
	for _, fieldName := range []string{"STTProfile", "TTSProfile"} {
		field, ok := reflect.TypeOf(TranslateVideoInput{}).FieldByName(fieldName)
		if !ok {
			t.Fatalf("expected %s field", fieldName)
		}
		if schema := field.Tag.Get("jsonschema"); !strings.Contains(schema, "xai") {
			t.Fatalf("expected %s schema to mention xai, got %q", fieldName, schema)
		}
	}
}

func TestTranslateVideoPostsGrokProfile(t *testing.T) {
	originalServerURL := serverURL
	originalHTTPClient := httpClient
	originalConf := config.Conf
	t.Cleanup(func() {
		serverURL = originalServerURL
		httpClient = originalHTTPClient
		config.Conf = originalConf
	})

	config.Conf.Models.LLM = map[string]config.ModelProfileConfig{
		"grok": {
			Provider: "xai-oauth",
			BaseURL:  "https://api.x.ai/v1",
			Model:    "grok-4.20-0309-non-reasoning",
		},
	}
	config.Conf.Models.STT = map[string]config.ModelProfileConfig{
		"default": {Provider: "openai"},
	}
	config.Conf.Models.TTS = map[string]config.ModelProfileConfig{
		"default": {Provider: "openai", Voices: []string{"Ryan"}},
	}

	var payload map[string]any
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/capability/subtitleTask" {
			t.Fatalf("expected subtitle task path, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":0,"data":{"task_id":"task-123"}}`))
	}))
	defer testServer.Close()

	serverURL = testServer.URL
	httpClient = testServer.Client()

	_, output, err := TranslateVideo(context.Background(), nil, TranslateVideoInput{
		URL:        "https://example.com/video.mp4",
		LLMProfile: "grok",
		TTS:        true,
	})
	if err != nil {
		t.Fatalf("TranslateVideo failed: %v", err)
	}
	if output.TaskID != "task-123" {
		t.Fatalf("expected task-123, got %q", output.TaskID)
	}
	if got := payload["llm_profile"]; got != "grok" {
		t.Fatalf("expected grok llm_profile, got %#v", got)
	}
}

func TestTranslateVideoDoesNotDefaultLLMProfile(t *testing.T) {
	originalServerURL := serverURL
	originalHTTPClient := httpClient
	t.Cleanup(func() {
		serverURL = originalServerURL
		httpClient = originalHTTPClient
	})

	var payload map[string]any
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":0,"data":{"task_id":"task-123"}}`))
	}))
	defer testServer.Close()

	serverURL = testServer.URL
	httpClient = testServer.Client()

	_, _, err := TranslateVideo(context.Background(), nil, TranslateVideoInput{
		URL: "https://example.com/video.mp4",
	})
	if err != nil {
		t.Fatalf("TranslateVideo failed: %v", err)
	}
	if got := payload["llm_profile"]; got != "" {
		t.Fatalf("expected empty llm_profile, got %#v", got)
	}
}

func TestTranslateVideoLeavesVoiceEmptyForServerRandomSelection(t *testing.T) {
	originalServerURL := serverURL
	originalHTTPClient := httpClient
	originalConf := config.Conf
	t.Cleanup(func() {
		serverURL = originalServerURL
		httpClient = originalHTTPClient
		config.Conf = originalConf
	})

	config.Conf.Models.TTS = map[string]config.ModelProfileConfig{
		"xai": {
			Provider: "xai-oauth",
			Voices:   []string{"eve", "ara"},
		},
	}

	var payload map[string]any
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":0,"data":{"task_id":"task-123"}}`))
	}))
	defer testServer.Close()

	serverURL = testServer.URL
	httpClient = testServer.Client()

	_, _, err := TranslateVideo(context.Background(), nil, TranslateVideoInput{
		URL:        "https://example.com/video.mp4",
		TTS:        true,
		TTSProfile: "xai",
	})
	if err != nil {
		t.Fatalf("TranslateVideo failed: %v", err)
	}
	if got := payload["tts_voice_code"]; got != "" {
		t.Fatalf("expected empty voice for server-side random selection, got %#v", got)
	}
}
