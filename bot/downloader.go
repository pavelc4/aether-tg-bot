package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pavelc4/aether-tg-bot/config"
)

func DownloadVideo(url string) ([]string, int64, string, error) {
	return runYTDLP(url, false)
}

func DownloadAudio(url string) ([]string, int64, string, error) {
	return runYTDLP(url, true)
}

func runYTDLP(url string, audioOnly bool) ([]string, int64, string, error) {

	filePaths, err := DownloadMediaWithCobalt(url, audioOnly)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to download with Cobalt: %w", err)
	}

	if len(filePaths) == 0 {
		return nil, 0, "", fmt.Errorf("Cobalt download returned no files")
	}

	var totalSize int64
	for _, path := range filePaths {
		if fileInfo, err := os.Stat(path); err == nil {
			totalSize += fileInfo.Size()
		}
	}

	provider := "Cobalt"

	return filePaths, totalSize, provider, nil
}

func DownloadMediaWithCobalt(mediaURL string, audioOnly bool) ([]string, error) {
	log.Printf("Attempting to download media from %s using Cobalt API", mediaURL)
	cobaltAPIURL := config.GetCobaltAPI()

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
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	log.Printf("Sending request to Cobalt API for URL: %s", mediaURL)

	req, err := http.NewRequest("POST", cobaltAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	apiKey := os.Getenv("COBALT_API_KEY")
	if apiKey != "" {
		req.Header.Set("Authorization", "Api-Key "+apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request to Cobalt API: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Received response from Cobalt API. Status Code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("Cobalt API non-OK response body: %s", string(bodyBytes))
		return nil, fmt.Errorf("Cobalt API returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cobalt API response body: %w", err)
	}
	log.Printf("Cobalt API response body: %s", string(bodyBytes))

	var cobaltResponse struct {
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

	if err := json.Unmarshal(bodyBytes, &cobaltResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Cobalt API response: %w", err)
	}
	log.Printf("Cobalt API response status: %s", cobaltResponse.Status)

	var downloadedFilePaths []string

	switch cobaltResponse.Status {
	case "tunnel", "redirect":
		if cobaltResponse.URL != "" {
			filePath, err := downloadDirectImage(cobaltResponse.URL, cobaltResponse.Filename)
			if err != nil {
				return nil, fmt.Errorf("failed to download media from Cobalt URL: %w", err)
			}
			downloadedFilePaths = append(downloadedFilePaths, filePath)
		} else {
			return nil, fmt.Errorf("Cobalt API returned tunnel/redirect status but no URL")
		}
	case "picker":
		if len(cobaltResponse.Picker) > 0 {
			for _, item := range cobaltResponse.Picker {
				if item.URL != "" {

					filePath, err := downloadDirectImage(item.URL, "")
					if err != nil {
						log.Printf("Warning: Failed to download one item from picker: %v", err)
						continue
					}
					downloadedFilePaths = append(downloadedFilePaths, filePath)
				}
			}
			if len(downloadedFilePaths) == 0 {
				return nil, fmt.Errorf("Cobalt API returned picker status but no downloadable URLs")
			}
		} else {
			return nil, fmt.Errorf("Cobalt API returned picker status but empty picker array")
		}
	case "error":
		return nil, fmt.Errorf("Cobalt API returned error: %s (context: %v)", cobaltResponse.Error.Code, cobaltResponse.Error.Context)
	default:
		return nil, fmt.Errorf("Cobalt API returned unknown status: %s", cobaltResponse.Status)
	}

	return downloadedFilePaths, nil
}

func downloadDirectImage(mediaURL string, suggestedFilename string) (string, error) {
	log.Printf("Attempting to download direct image from URL: %s (suggested filename: %s)", mediaURL, suggestedFilename)
	tmpDir, err := os.MkdirTemp("", "aether-scrape-")
	if err != nil {
		return "", err
	}
	log.Printf("Created temporary directory: %s", tmpDir)

	resp, err := http.Get(mediaURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ext := ".tmp"
	if suggestedFilename != "" {
		ext = filepath.Ext(suggestedFilename)
		if ext == "" {
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "image/png") {
				ext = ".png"
			} else if strings.Contains(contentType, "image/gif") {
				ext = ".gif"
			} else if strings.Contains(contentType, "video/mp4") {
				ext = ".mp4"
			} else if strings.Contains(contentType, "video/webm") {
				ext = ".webm"
			} else if strings.Contains(contentType, "video/quicktime") {
				ext = ".mov"
			} else if strings.Contains(contentType, "image/jpeg") {
				ext = ".jpg"
			} else {
				parsedURL, parseErr := url.Parse(mediaURL)
				if parseErr == nil {
					pathExt := filepath.Ext(parsedURL.Path)
					if pathExt != "" {
						ext = pathExt
					}
				}
			}
		}
	} else {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "image/png") {
			ext = ".png"
		} else if strings.Contains(contentType, "image/gif") {
			ext = ".gif"
		} else if strings.Contains(contentType, "video/mp4") {
			ext = ".mp4"
		} else if strings.Contains(contentType, "video/webm") {
			ext = ".webm"
		} else if strings.Contains(contentType, "video/quicktime") {
			ext = ".mov"
		} else if strings.Contains(contentType, "image/jpeg") {
			ext = ".jpg"
		} else {
			parsedURL, parseErr := url.Parse(mediaURL)
			if parseErr == nil {
				pathExt := filepath.Ext(parsedURL.Path)
				if pathExt != "" {
					ext = pathExt
				}
			}
		}
	}

	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}
	log.Printf("Successfully downloaded file to: %s", filePath)

	return filePath, nil
}
