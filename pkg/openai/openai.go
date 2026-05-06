package openai

import (
	"context"
	"encoding/json"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"io"
	"krillin-ai/config"
	"krillin-ai/log"
	"net/http"
	"os"
	"strings"
)

func (c *Client) ChatCompletion(query string) (string, error) {
	var responseFormat *openai.ChatCompletionResponseFormat

	req := openai.ChatCompletionRequest{
		Model: config.Conf.Llm.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are an assistant that helps with subtitle translation.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: query,
			},
		},
		Temperature:    0.9,
		Stream:         false,
		MaxTokens:      8192,
		ResponseFormat: responseFormat,
	}

	resp, err := c.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		log.GetLogger().Error("openai create chat completion failed", zap.Error(err))
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *Client) Text2Speech(text, voice string, outputFile string) error {
	baseUrl := config.Conf.Tts.Openai.BaseUrl
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	url := baseUrl + "/audio/speech"

	// 创建HTTP请求
	reqBody := fmt.Sprintf(`{
		"model": "%s",
		"input": "%s",
		"voice":"%s",
		"response_format": "wav"
	}`, config.Conf.Tts.Openai.Model, text, voice)
	req, err := http.NewRequest("POST", url, strings.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Conf.Tts.Openai.ApiKey))

	// 发送HTTP请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.GetLogger().Error("openai tts failed", zap.Int("status_code", resp.StatusCode), zap.String("body", string(body)))
		return fmt.Errorf("openai tts none-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 檢查是否為 JSON 回應（aiark 格式）
	var ttsResponse struct {
		File string `json:"file"`
	}
	if json.Unmarshal(body, &ttsResponse) == nil && ttsResponse.File != "" {
		// aiark 返回 JSON，需要下載音訊
		audioUrl := strings.Replace(baseUrl, "/v1", "", 1) + ttsResponse.File
		audioReq, err := http.NewRequest("GET", audioUrl, nil)
		if err != nil {
			return err
		}
		audioReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Conf.Tts.Openai.ApiKey))

		audioResp, err := client.Do(audioReq)
		if err != nil {
			return err
		}
		defer audioResp.Body.Close()

		if audioResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download audio: %d", audioResp.StatusCode)
		}

		file, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, audioResp.Body)
		return err
	}

	// 標準回應（直接是音訊資料）
	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(body)
	return err
}

func parseJSONResponse(jsonStr string) (string, error) {
	var response struct {
		Translations []struct {
			Original   string `json:"original_sentence"`
			Translated string `json:"translated_sentence"`
		} `json:"translations"`
	}

	err := json.Unmarshal([]byte(jsonStr), &response)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	var result strings.Builder
	for i, item := range response.Translations {
		result.WriteString(fmt.Sprintf("%d\n%s\n%s\n\n",
			i+1,
			item.Translated,
			item.Original))
	}

	return result.String(), nil
}
