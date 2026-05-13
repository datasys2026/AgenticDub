package openai

import (
	"encoding/json"
	"fmt"
	"io"
	"krillin-ai/config"
	"krillin-ai/log"
	"net/http"
	"os"
	"strings"

	"go.uber.org/zap"
)

func (c *Client) Text2Speech(text, voice string, outputFile string) error {
	baseUrl := config.Conf.Tts.Openai.BaseUrl
	if baseUrl == "" {
		baseUrl = "https://api.openai.com/v1"
	}
	url := baseUrl + "/audio/speech"

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

	var ttsResponse struct {
		File string `json:"file"`
	}
	if json.Unmarshal(body, &ttsResponse) == nil && ttsResponse.File != "" {
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

	file, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(body)
	return err
}