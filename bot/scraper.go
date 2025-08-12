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
)

func DownloadMediaWithCobalt(mediaURL string) ([]string, error) {
	log.Printf("Attempting to download media from %s using Cobalt API", mediaURL)
	cobaltAPIURL := "http://localhost:8080/" // Using your self-hosted Cobalt instance

	requestBody := map[string]interface{}{
		"url":          mediaURL,
		"downloadMode": "auto", // "auto" for both video and photo
		"videoQuality": "max",  // Max quality for video
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

	client := &http.Client{Timeout: 30 * time.Second} // Set a timeout for the HTTP request
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

	// Read the response body once for logging and decoding
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

	if err := json.Unmarshal(bodyBytes, &cobaltResponse); err != nil { // Use Unmarshal with bodyBytes
		return nil, fmt.Errorf("failed to decode Cobalt API response: %w", err)
	}
	log.Printf("Cobalt API response status: %s", cobaltResponse.Status)

	var downloadedFilePaths []string

	switch cobaltResponse.Status {
	case "tunnel", "redirect":
		if cobaltResponse.URL != "" {
			filePath, err := downloadDirectImage(cobaltResponse.URL, cobaltResponse.Filename) // Pass filename
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

	ext := ".tmp" // Default to a generic temporary extension
	if suggestedFilename != "" {
		ext = filepath.Ext(suggestedFilename)
		if ext == "" { // If suggested filename has no extension, try Content-Type
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "image/png") {
				ext = ".png"
			} else if strings.Contains(contentType, "image/gif") {
				ext = ".gif"
			} else if strings.Contains(contentType, "video/mp4") {
				ext = ".mp4"
			} else if strings.Contains(contentType, "video/webm") {
				ext = ".webm"
			} else if strings.Contains(contentType, "video/quicktime") { // For .mov
				ext = ".mov"
			} else if strings.Contains(contentType, "image/jpeg") {
				ext = ".jpg"
			} else {
				// Fallback: try to get extension from the URL path
				parsedURL, parseErr := url.Parse(mediaURL)
				if parseErr == nil {
					pathExt := filepath.Ext(parsedURL.Path)
					if pathExt != "" {
						ext = pathExt
					}
				}
			}
		}
	} else { // No suggested filename, rely on Content-Type or URL
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "image/png") {
			ext = ".png"
		} else if strings.Contains(contentType, "image/gif") {
			ext = ".gif"
		} else if strings.Contains(contentType, "video/mp4") {
			ext = ".mp4"
		} else if strings.Contains(contentType, "video/webm") {
			ext = ".webm"
		} else if strings.Contains(contentType, "video/quicktime") { // For .mov
			ext = ".mov"
		} else if strings.Contains(contentType, "image/jpeg") {
			ext = ".jpg"
		} else {
			// Fallback: try to get extension from the URL path
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
