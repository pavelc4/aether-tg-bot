package downloader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/pavelc4/aether-tg-bot/config"
)

type cobaltAPIResponse struct {
	Status   string `json:"status"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Picker   []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"picker"`
	Error struct {
		Code    string      `json:"code"`
		Context interface{} `json:"context"`
	} `json:"error"`
}

func DownloadMediaWithCobalt(mediaURL string, audioOnly bool) ([]string, error) {
	response, err := requestCobaltAPI(mediaURL, audioOnly)
	if err != nil {
		return nil, err
	}
	return processCobaltResponse(response)
}

func requestCobaltAPI(mediaURL string, audioOnly bool) (*cobaltAPIResponse, error) {
	requestBody := map[string]interface{}{
		"url":          mediaURL,
		"downloadMode": "auto",
		"videoQuality": "max",
	}

	if audioOnly {
		requestBody["downloadMode"] = "audio"
		requestBody["audioFormat"] = "mp3"
	}

	if strings.Contains(mediaURL, "tiktok.com") {
		requestBody["allowH265"] = true
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cobaltTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", config.GetCobaltAPI(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if apiKey := os.Getenv("COBALT_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Api-Key "+apiKey)
	}

	resp, err := GetDownloadClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("cobalt request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cobalt returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var cobaltResponse cobaltAPIResponse
	if err := json.Unmarshal(bodyBytes, &cobaltResponse); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &cobaltResponse, nil
}

func processCobaltResponse(response *cobaltAPIResponse) ([]string, error) {
	switch response.Status {
	case "tunnel", "redirect":
		return handleTunnelRedirect(response)
	case "picker":
		return handlePicker(response)
	case "error":
		return nil, fmt.Errorf("cobalt error: %s (context: %v)",
			response.Error.Code, response.Error.Context)
	default:
		return nil, fmt.Errorf("unknown cobalt status: %s", response.Status)
	}
}

func handleTunnelRedirect(response *cobaltAPIResponse) ([]string, error) {
	if response.URL == "" {
		return nil, fmt.Errorf("empty URL in tunnel/redirect response")
	}

	filePath, err := downloadFile(response.URL, response.Filename)
	if err != nil {
		return nil, fmt.Errorf("download from cobalt URL failed: %w", err)
	}

	return []string{filePath}, nil
}

func handlePicker(response *cobaltAPIResponse) ([]string, error) {
	if len(response.Picker) == 0 {
		return nil, fmt.Errorf("empty picker array")
	}

	var filePaths []string
	for _, item := range response.Picker {
		if item.URL == "" {
			continue
		}

		filePath, err := downloadFile(item.URL, "")
		if err != nil {
			log.Printf("Failed to download picker item: %v", err)
			continue
		}
		filePaths = append(filePaths, filePath)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files downloaded from picker")
	}

	return filePaths, nil
}
