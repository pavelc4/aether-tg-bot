package bot

import (
	"bytes"
	"context"
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

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableKeepAlives:     false,
	},
}

// Content type mappings
var (
	contentTypeToExt = map[string]string{
		"image/png":        ".png",
		"image/gif":        ".gif",
		"image/jpeg":       ".jpg",
		"video/mp4":        ".mp4",
		"video/webm":       ".webm",
		"video/quicktime":  ".mov",
		"video/x-matroska": ".mkv",
		"audio/mpeg":       ".mp3",
	}

	imageContentTypes = map[string]string{
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/jpeg": ".jpg",
	}
)

const (
	minFileSize     = 5120 // 5KB minimum
	downloadTimeout = 60 * time.Second
	cobaltTimeout   = 30 * time.Second
	ytdlpTimeout    = 5 * time.Minute
)

func CleanupTempFiles(filePaths []string) {
	for _, path := range filePaths {
		if path == "" {
			continue
		}

		dir := filepath.Dir(path)
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("Warning: Failed to cleanup temp directory %s: %v", dir, err)
			continue
		}
		log.Printf("Cleaned up temp directory: %s", dir)
	}
}

func DownloadVideo(url string) ([]string, int64, string, error) {
	return runYTDLP(url, false)
}

func DownloadAudio(url string) ([]string, int64, string, error) {
	return runYTDLP(url, true)
}

func runYTDLP(url string, audioOnly bool) ([]string, int64, string, error) {
	filePaths, err := DownloadMediaWithCobalt(url, audioOnly)
	if err == nil {
		return calculateTotalSize(filePaths, "Cobalt")
	}

	log.Printf("Cobalt download failed: %v", err)

	// Early return if not YouTube
	if !isYouTubeURL(url) {
		return nil, 0, "", fmt.Errorf("failed to download with Cobalt: %w", err)
	}

	log.Printf("YouTube link detected. Falling back to yt-dlp.")
	filePaths, err = DownloadMediaWithYTDLP(url, audioOnly)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to download with Cobalt and yt-dlp: %w", err)
	}

	return calculateTotalSize(filePaths, "yt-dlp")
}

// Helper: Check if URL is YouTube
func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

// Helper: Calculate total size
func calculateTotalSize(filePaths []string, provider string) ([]string, int64, string, error) {
	if len(filePaths) == 0 {
		return nil, 0, "", fmt.Errorf("download returned no files")
	}

	var totalSize int64
	for _, path := range filePaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			continue
		}
		totalSize += fileInfo.Size()
	}

	return filePaths, totalSize, provider, nil
}

func DownloadMediaWithYTDLP(mediaURL string, audioOnly bool) ([]string, error) {
	log.Printf("Attempting to download media from %s using yt-dlp.", mediaURL)

	tmpDir, err := os.MkdirTemp("", "aether-ytdlp-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpTimeout)
	defer cancel()

	format := getYTDLPFormat(audioOnly)
	cmd := exec.CommandContext(ctx, "yt-dlp", "-f", format, "-o", filepath.Join(tmpDir, "%(title)s.%(ext)s"), mediaURL)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp execution failed: %w\n%s", err, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to list downloaded files: %w", err)
	}

	if len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp downloaded no files")
	}

	return files, nil
}

// Helper: Get yt-dlp format string
func getYTDLPFormat(audioOnly bool) string {
	if audioOnly {
		return "bestaudio/best"
	}
	return "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best"
}

func DownloadMediaWithCobalt(mediaURL string, audioOnly bool) ([]string, error) {
	log.Printf("Attempting to download media from %s using Cobalt API.", mediaURL)

	cobaltResponse, err := requestCobaltAPI(mediaURL, audioOnly)
	if err != nil {
		return nil, err
	}

	return processCobaltResponse(cobaltResponse)
}

// Helper: Request Cobalt API
func requestCobaltAPI(mediaURL string, audioOnly bool) (*cobaltAPIResponse, error) {
	cobaltAPIURL := config.GetCobaltAPI()

	requestBody := buildCobaltRequest(mediaURL, audioOnly)
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	log.Printf("Sending request to Cobalt API for URL: %s", mediaURL)

	ctx, cancel := context.WithTimeout(context.Background(), cobaltTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", cobaltAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	if apiKey := os.Getenv("COBALT_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Api-Key "+apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request to Cobalt API: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	log.Printf("Received response from Cobalt API. Status Code: %d", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Cobalt API response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Cobalt API non-OK response body: %s", string(bodyBytes))
		return nil, fmt.Errorf("Cobalt API returned non-OK status: %d", resp.StatusCode)
	}

	log.Printf("Cobalt API response body: %s", string(bodyBytes))

	var cobaltResponse cobaltAPIResponse
	if err := json.Unmarshal(bodyBytes, &cobaltResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Cobalt API response: %w", err)
	}

	log.Printf("Cobalt API response status: %s", cobaltResponse.Status)
	return &cobaltResponse, nil
}

// Type definitions
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

// Helper: Build Cobalt request body
func buildCobaltRequest(mediaURL string, audioOnly bool) map[string]interface{} {
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

	return requestBody
}

// Helper: Process Cobalt response
func processCobaltResponse(response *cobaltAPIResponse) ([]string, error) {
	switch response.Status {
	case "tunnel", "redirect":
		return handleTunnelRedirect(response)
	case "picker":
		return handlePicker(response)
	case "error":
		return nil, fmt.Errorf("Cobalt API returned error: %s (context: %v)",
			response.Error.Code, response.Error.Context)
	default:
		return nil, fmt.Errorf("Cobalt API returned unknown status: %s", response.Status)
	}
}

// Helper: Handle tunnel/redirect
func handleTunnelRedirect(response *cobaltAPIResponse) ([]string, error) {
	if response.URL == "" {
		return nil, fmt.Errorf("Cobalt API returned tunnel/redirect status but no URL")
	}

	log.Printf("Cobalt API response URL: %s", response.URL)
	filePath, err := downloadFile(response.URL, response.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to download media from Cobalt URL: %w", err)
	}

	return []string{filePath}, nil
}

// Helper: Handle picker
func handlePicker(response *cobaltAPIResponse) ([]string, error) {
	if len(response.Picker) == 0 {
		return nil, fmt.Errorf("Cobalt API returned picker status but empty picker array")
	}

	log.Printf("Cobalt API response Picker: %+v", response.Picker)

	var downloadedFilePaths []string
	for _, item := range response.Picker {
		if item.URL == "" {
			continue
		}

		filePath, err := downloadFile(item.URL, "")
		if err != nil {
			log.Printf("Warning: Failed to download one item from picker: %v", err)
			continue
		}
		downloadedFilePaths = append(downloadedFilePaths, filePath)
	}

	if len(downloadedFilePaths) == 0 {
		return nil, fmt.Errorf("Cobalt API returned picker status but no downloadable URLs")
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

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	ext := determineFileExtension(resp, mediaURL, suggestedFilename)
	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	file, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	if size < minFileSize {
		log.Printf("Downloaded file is too small (%d bytes), likely invalid. Deleting file: %s", size, filePath)
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("downloaded file too small (%d bytes)", size)
	}

	log.Printf("Successfully downloaded file to: %s (Size: %d bytes)", filePath, size)
	return filePath, nil
}

// Helper: Determine file extension
func determineFileExtension(resp *http.Response, mediaURL, suggestedFilename string) string {
	// Priority 1: Suggested filename
	if suggestedFilename != "" {
		if ext := filepath.Ext(suggestedFilename); ext != "" {
			return ext
		}
	}

	// Priority 2: Content-Type header
	contentType := resp.Header.Get("Content-Type")
	for ct, ext := range contentTypeToExt {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}

	// Priority 3: URL path extension
	if parsedURL, err := url.Parse(mediaURL); err == nil {
		if ext := filepath.Ext(parsedURL.Path); ext != "" {
			return ext
		}
	}

	return ".tmp"
}

func DownloadImage(imageUrl string) (string, int64, error) {
	log.Printf("Attempting to download image from URL: %s", imageUrl)

	tmpDir, err := os.MkdirTemp("", "aether-image-")
	if err != nil {
		return "", 0, err
	}
	log.Printf("Created temporary directory for image: %s", tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), cobaltTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", imageUrl, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", 0, fmt.Errorf("failed to download image: status code %d", resp.StatusCode)
	}

	ext := getImageExtension(resp.Header.Get("Content-Type"))
	filePath := filepath.Join(tmpDir, fmt.Sprintf("%d%s", time.Now().UnixNano(), ext))

	file, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}

	if size < minFileSize {
		log.Printf("Downloaded image is too small (%d bytes), likely invalid. Deleting file: %s", size, filePath)
		os.RemoveAll(tmpDir)
		return "", 0, fmt.Errorf("downloaded image too small (%d bytes)", size)
	}

	log.Printf("Successfully downloaded image to: %s (Size: %d bytes)", filePath, size)
	return filePath, size, nil
}

// Helper: Get image extension from content type
func getImageExtension(contentType string) string {
	for ct, ext := range imageContentTypes {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}
	return ".jpg" // default
}
