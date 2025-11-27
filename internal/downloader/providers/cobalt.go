package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pavelc4/aether-tg-bot/config"
	httpclient "github.com/pavelc4/aether-tg-bot/pkg/http"
)

const (
	cobaltTimeout   = 60 * time.Second
	downloadTimeout = 2 * time.Minute
	minFileSize     = 5120
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

type CobaltProvider struct {
	timeout  time.Duration
	client   *http.Client
	handlers map[string]responseHandler
}

func NewCobaltProvider() *CobaltProvider {
	cp := &CobaltProvider{
		timeout: cobaltTimeout,
		client:  httpclient.GetDownloadClient(),
	}
	cp.handlers = map[string]responseHandler{
		"tunnel":   cp.handleTunnelRedirect,
		"redirect": cp.handleTunnelRedirect,
		"picker":   cp.handlePicker,
		"error":    cp.handleError,
	}
	return cp
}

type responseHandler func(context.Context, *cobaltAPIResponse) ([]string, error)

func (cp *CobaltProvider) Name() string {
	return "Cobalt"
}

func (cp *CobaltProvider) CanHandle(url string) bool {
	return true
}

func (cp *CobaltProvider) Download(ctx context.Context, url string, audioOnly bool) ([]string, error) {
	log.Printf(" Cobalt: Requesting... (audio=%v)", audioOnly)

	response, err := cp.requestAPI(ctx, url, audioOnly)
	if err != nil {
		return nil, fmt.Errorf("cobalt API request failed: %w", err)
	}

	return cp.processResponse(ctx, response)
}

func (cp *CobaltProvider) requestAPI(ctx context.Context, mediaURL string, audioOnly bool) (*cobaltAPIResponse, error) {
	requestBody := map[string]interface{}{
		"url":          mediaURL,
		"downloadMode": "auto",
		"videoQuality": "max",
	}

	if audioOnly {
		requestBody["downloadMode"] = "audio"
		requestBody["audioFormat"] = "mp3"
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.GetCobaltAPI(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := cp.client.Do(req)
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

func (cp *CobaltProvider) processResponse(ctx context.Context, response *cobaltAPIResponse) ([]string, error) {
	if handler, exists := cp.handlers[response.Status]; exists {
		return handler(ctx, response)
	}
	return nil, fmt.Errorf("unknown cobalt status: %s", response.Status)
}

func (cp *CobaltProvider) handleError(ctx context.Context, response *cobaltAPIResponse) ([]string, error) {
	return nil, fmt.Errorf("cobalt error: %s", response.Error.Code)
}

func (cp *CobaltProvider) handleTunnelRedirect(ctx context.Context, response *cobaltAPIResponse) ([]string, error) {
	if response.URL == "" {
		return nil, fmt.Errorf("empty URL in response")
	}

	log.Printf("Cobalt: Downloading from redirect URL")
	filePath, err := cp.downloadFile(ctx, response.URL, response.Filename)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	return []string{filePath}, nil
}

func (cp *CobaltProvider) handlePicker(ctx context.Context, response *cobaltAPIResponse) ([]string, error) {
	if len(response.Picker) == 0 {
		return nil, fmt.Errorf("empty picker array")
	}

	var filePaths []string
	for i, item := range response.Picker {
		if item.URL == "" {
			continue
		}

		log.Printf("Cobalt: Downloading picker item %d/%d", i+1, len(response.Picker))
		filePath, err := cp.downloadFile(ctx, item.URL, "")
		if err != nil {
			log.Printf("⚠️ Failed to download item %d: %v", i+1, err)
			continue
		}

		filePaths = append(filePaths, filePath)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no files downloaded")
	}

	return filePaths, nil
}

func (cp *CobaltProvider) downloadFile(ctx context.Context, mediaURL, suggestedFilename string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "aether-cobalt-")
	if err != nil {
		return "", fmt.Errorf("create temp dir failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := cp.client.Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ext := cp.getExtension(resp.Header.Get("Content-Type"), suggestedFilename)
	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	file, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("save file failed: %w", err)
	}

	if size < minFileSize {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("file too small: %d bytes", size)
	}

	return filePath, nil
}

func (cp *CobaltProvider) getExtension(contentType, suggestedFilename string) string {
	if ext := filepath.Ext(suggestedFilename); ext != "" {
		return ext
	}

	contentTypeMap := map[string]string{
		"video/mp4":  ".mp4",
		"video/webm": ".webm",
		"audio/mpeg": ".mp3",
		"audio/mp4":  ".m4a",
		"image/jpeg": ".jpg",
		"image/png":  ".png",
	}

	for ct, ext := range contentTypeMap {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}

	return ".mp4"
}
