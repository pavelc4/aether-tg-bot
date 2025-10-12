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
	"os/exec"
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
	provider := "Cobalt"

	if err != nil {
		log.Printf("Cobalt download failed: %v", err)
		if !strings.Contains(url, "youtube.com") && !strings.Contains(url, "youtu.be") {
			return nil, 0, "", fmt.Errorf("failed to download with Cobalt: %w", err)
		}

		log.Printf("YouTube link detected. Falling back to yt-dlp.")
		filePaths, err = DownloadMediaWithYTDLP(url, audioOnly)
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to download with Cobalt and yt-dlp: %w", err)
		}
		provider = "yt-dlp"
	}

	if len(filePaths) == 0 {
		return nil, 0, "", fmt.Errorf("download returned no files")
	}

	var totalSize int64
	for _, path := range filePaths {
		if fileInfo, err := os.Stat(path); err == nil {
			totalSize += fileInfo.Size()
		}
	}

	return filePaths, totalSize, provider, nil
}

func DownloadMediaWithYTDLP(mediaURL string, audioOnly bool) ([]string, error) {
	log.Printf("Attempting to download media from %s using yt-dlp.", mediaURL)

	tmpDir, err := os.MkdirTemp("", "aether-ytdlp-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	format := "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best"
	if audioOnly {
		format = "bestaudio/best"
	}

	cmd := exec.Command("yt-dlp", "-f", format, "-o", filepath.Join(tmpDir, "% (title)s.%(ext)s"), mediaURL)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp execution failed: %w\n%s", err, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil {
		return nil, fmt.Errorf("failed to list downloaded files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("yt-dlp downloaded no files")
	}

	return files, nil
}


func DownloadMediaWithCobalt(mediaURL string, audioOnly bool) ([]string, error) {
	log.Printf("Attempting to download media from %s using Cobalt API. mediaURL: %s", mediaURL, mediaURL)
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

	if cobaltResponse.Status == "tunnel" || cobaltResponse.Status == "redirect" {
		log.Printf("Cobalt API response URL: %s", cobaltResponse.URL)
	} else if cobaltResponse.Status == "picker" {
		log.Printf("Cobalt API response Picker: %+v", cobaltResponse.Picker)
	} else if cobaltResponse.Status == "error" {
		log.Printf("Cobalt API returned error status. Error Code: %s, Context: %+v, Full Response: %+v", cobaltResponse.Error.Code, cobaltResponse.Error.Context, cobaltResponse)
	}

	var downloadedFilePaths []string

	switch cobaltResponse.Status {
	case "tunnel", "redirect":
		if cobaltResponse.URL != "" {
			filePath, err := downloadFile(cobaltResponse.URL, cobaltResponse.Filename)
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

					filePath, err := downloadFile(item.URL, "")
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

func downloadFile(mediaURL string, suggestedFilename string) (string, error) {
	log.Printf("Attempting to download file from URL: %s (suggested filename: %s)", mediaURL, suggestedFilename)
	tmpDir, err := os.MkdirTemp("", "aether-scrape-")
	if err != nil {
		return "", err
	}
	log.Printf("Created temporary directory: %s", tmpDir)

	client := &http.Client{}
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	ext := ".tmp"
	if suggestedFilename != "" {
		ext = filepath.Ext(suggestedFilename)
	}

	if ext == "" || ext == ".tmp" {
		contentType := resp.Header.Get("Content-Type")
		contentTypeMap := map[string]string{
			"image/png":  ".png",
			"image/gif":  ".gif",
			"image/jpeg": ".jpg",
			"video/mp4":  ".mp4",
			"video/webm": ".webm",
			"video/quicktime": ".mov",
			"video/x-matroska": ".mkv",
			"audio/mpeg": ".mp3",
		}
		for ct, e := range contentTypeMap {
			if strings.Contains(contentType, ct) {
				ext = e
				break
			}
		}
	}

	if ext == "" || ext == ".tmp" {
		parsedURL, parseErr := url.Parse(mediaURL)
		if parseErr == nil {
			pathExt := filepath.Ext(parsedURL.Path)
			if pathExt != "" {
				ext = pathExt
			}
		}
	}

	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", err
	}

	if size < 5120 {
		log.Printf("Downloaded file is too small (%d bytes), likely invalid. Deleting file: %s", size, filePath)
		os.Remove(filePath)
		return "", fmt.Errorf("downloaded file too small (%d bytes)", size)
	}

	log.Printf("Successfully downloaded file to: %s (Size: %d bytes)", filePath, size)

	return filePath, nil
}

func DownloadImage(imageUrl string) (string, int64, error) {
	log.Printf("Attempting to download image from URL: %s", imageUrl)
	tmpDir, err := os.MkdirTemp("", "aether-image-")
	if err != nil {
		return "", 0, err
	}
	log.Printf("Created temporary directory for image: %s", tmpDir)

	resp, err := http.Get(imageUrl)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "image/png") {
		ext = ".png"
	} else if strings.Contains(contentType, "image/gif") {
		ext = ".gif"
	} else if strings.Contains(contentType, "image/jpeg") {
		ext = ".jpg"
	}

	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))
	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", 0, err
	}

	if size < 5120 {
		log.Printf("Downloaded image is too small (%d bytes), likely invalid. Deleting file: %s", size, filePath)
		os.Remove(filePath)
		return "", 0, fmt.Errorf("downloaded image too small (%d bytes)", size)
	}

	log.Printf("Successfully downloaded image to: %s (Size: %d bytes)", filePath, size)

	return filePath, size, nil
}
