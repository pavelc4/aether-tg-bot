package bot

import (
	"bufio"
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
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/config"
)

// HTTP client untuk Telegram Bot API (long polling)
var botClient = &http.Client{
	Timeout: 90 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 60 * time.Second,
		DisableKeepAlives:     false,
	},
}

// HTTP client untuk download (Cobalt, yt-dlp, images)
var downloadClient = &http.Client{
	Timeout: 10 * time.Minute,
	Transport: &http.Transport{
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		MaxConnsPerHost:       30,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 90 * time.Second,
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

var (
	// Regex untuk parse yt-dlp progress
	ytdlpProgressRegex = regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)
)

const (
	minFileSize     = 5120             // 5KB minimum
	downloadTimeout = 2 * time.Minute  // HTTP download timeout
	cobaltTimeout   = 60 * time.Second // Cobalt API timeout
	ytdlpTimeout    = 10 * time.Minute // yt-dlp execution timeout
)

// GetBotClient returns HTTP client untuk Telegram Bot API
func GetBotClient() *http.Client {
	return botClient
}

// CleanupTempFiles removes temporary directories safely
func CleanupTempFiles(filePaths []string) {
	for _, path := range filePaths {
		if path == "" {
			continue
		}

		dir := filepath.Dir(path)
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("Warning: Failed to cleanup temp directory %s: %v", dir, err)
		}
	}
}

func DownloadVideo(url string) ([]string, int64, string, error) {
	return runYTDLP(url, false)
}

func DownloadAudio(url string) ([]string, int64, string, error) {
	return runYTDLP(url, true)
}

func runYTDLP(url string, audioOnly bool) ([]string, int64, string, error) {
	// Try Cobalt first
	if filePaths, err := DownloadMediaWithCobalt(url, audioOnly); err == nil {
		return calculateTotalSize(filePaths, "Cobalt")
	}

	// Fallback to yt-dlp for YouTube
	if !isYouTubeURL(url) {
		return nil, 0, "", fmt.Errorf("unsupported platform or download failed")
	}

	log.Printf("Falling back to yt-dlp for YouTube URL")

	// Try without cookies first (with adaptive aria2c)
	filePaths, err := DownloadMediaWithYTDLP(url, audioOnly, false)
	if err != nil {
		log.Printf("yt-dlp without cookies failed, retrying with cookies...")
		filePaths, err = DownloadMediaWithYTDLP(url, audioOnly, true)
		if err != nil {
			return nil, 0, "", fmt.Errorf("all download methods failed: %w", err)
		}
	}

	// Set provider name based on adaptive mode
	provider := "yt-dlp"
	if GetCPUManager().IsEnabled() {
		provider = "yt-dlp (adaptive)"
	}

	return calculateTotalSize(filePaths, provider)
}

func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

// calculateTotalSize sums up file sizes and validates
func calculateTotalSize(filePaths []string, provider string) ([]string, int64, string, error) {
	if len(filePaths) == 0 {
		return nil, 0, "", fmt.Errorf("no files downloaded")
	}

	var totalSize int64
	for _, path := range filePaths {
		if info, err := os.Stat(path); err == nil {
			totalSize += info.Size()
		}
	}

	return filePaths, totalSize, provider, nil
}

// getCookiePath returns cookie file path from environment or default
func getCookiePath() string {
	if path := os.Getenv("YTDLP_COOKIES"); path != "" {
		return path
	}
	return "cookies.txt"
}

// DownloadMediaWithYTDLPWithProgress downloads media using yt-dlp with adaptive aria2c and progress tracking
func DownloadMediaWithYTDLPWithProgress(mediaURL string, audioOnly, useCookies bool, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, error) {
	useAria2 := isAria2Available()

	log.Printf("🚀 Starting download: url=%s, audio=%v, cookies=%v, aria2=%v, adaptive=%v",
		mediaURL, audioOnly, useCookies, useAria2, GetCPUManager().IsEnabled())

	tmpDir, err := os.MkdirTemp("", "aether-ytdlp-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpTimeout)
	defer cancel()

	// Start CPU monitoring in background if bot is provided
	if bot != nil && GetCPUManager().IsEnabled() {
		go GetCPUManager().MonitorCPUDuringDownload(ctx, bot, chatID, msgID)
	}

	// Build yt-dlp args with adaptive aria2c
	args := buildYTDLPArgsAdaptive(ctx, tmpDir, audioOnly, useCookies, useAria2)

	// Add progress flag for yt-dlp
	args = append(args, "--newline", "--progress")
	args = append(args, mediaURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	// Capture stdout for progress parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// Track progress in goroutine
	go trackYTDLPProgress(stdout, bot, chatID, msgID)

	if err := cmd.Wait(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil || len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no files downloaded")
	}

	log.Printf("✅ Downloaded %d file(s) to %s", len(files), tmpDir)
	return files, nil
}

// trackYTDLPProgress parses yt-dlp output and updates progress
func trackYTDLPProgress(stdout io.Reader, bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	scanner := bufio.NewScanner(stdout)
	lastUpdate := time.Now()
	updateInterval := 2 * time.Second // Update every 2 seconds

	for scanner.Scan() {
		line := scanner.Text()

		// Parse progress line
		if matches := ytdlpProgressRegex.FindStringSubmatch(line); len(matches) == 5 {
			// Only update if enough time has passed
			if time.Since(lastUpdate) < updateInterval {
				continue
			}

			percentage, _ := strconv.ParseFloat(matches[1], 64)

			progress := DownloadProgress{
				Percentage: percentage,
				Downloaded: matches[2],
				Speed:      matches[3],
				ETA:        matches[4],
				Status:     "Downloading",
			}

			UpdateProgressMessage(bot, chatID, msgID, "YouTube", progress)
			lastUpdate = time.Now()
		}
	}
}

// Wrapper untuk backward compatibility
func DownloadMediaWithYTDLP(mediaURL string, audioOnly, useCookies bool) ([]string, error) {
	return DownloadMediaWithYTDLPWithProgress(mediaURL, audioOnly, useCookies, nil, 0, 0)
}

// buildYTDLPArgsAdaptive constructs yt-dlp command arguments with adaptive aria2c
func buildYTDLPArgsAdaptive(ctx context.Context, tmpDir string, audioOnly, useCookies, useAria2 bool) []string {
	format := "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best"
	if audioOnly {
		format = "bestaudio/best"
	}

	args := []string{
		"-f", format,
		"-o", filepath.Join(tmpDir, "%(title)s.%(ext)s"),
		"--no-playlist",
		"--socket-timeout", "60",
		"--retries", "5",
		"--fragment-retries", "5",
		"--retry-sleep", "3",
	}

	// Add adaptive aria2c if available and enabled
	if useAria2 && GetCPUManager().IsEnabled() {
		aria2Args := GetCPUManager().BuildAria2Args(ctx)
		connections := GetCPUManager().GetOptimalConnections(ctx)

		args = append(args,
			"--external-downloader", "aria2c",
			"--external-downloader-args", aria2Args,
			"--concurrent-fragments", fmt.Sprintf("%d", maxInt(1, connections/4)),
		)
		log.Printf("✨ Using adaptive aria2c")
	} else if useAria2 {
		// Fallback to static aria2c config if adaptive disabled
		args = append(args,
			"--external-downloader", "aria2c",
			"--external-downloader-args", "-c -x 16 -s 16 -k 1M --file-allocation=none",
			"--concurrent-fragments", "4",
		)
		log.Printf("Using aria2c with 16 connections (static)")
	}

	// Add cookies if requested
	if useCookies {
		cookiePath := getCookiePath()
		if _, err := os.Stat(cookiePath); err == nil {
			args = append(args, "--cookies", cookiePath)
			log.Printf("🍪 Using cookies from: %s", cookiePath)
		} else {
			log.Printf("Cookie file not found at %s", cookiePath)
		}
	}

	return args
}

// Helper function for max (for Go < 1.21 compatibility)
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DownloadMediaWithCobalt downloads media using Cobalt API
func DownloadMediaWithCobalt(mediaURL string, audioOnly bool) ([]string, error) {
	response, err := requestCobaltAPI(mediaURL, audioOnly)
	if err != nil {
		return nil, err
	}
	return processCobaltResponse(response)
}

// requestCobaltAPI sends request to Cobalt API
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

	resp, err := downloadClient.Do(req)
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

// processCobaltResponse handles different Cobalt response types
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

// handleTunnelRedirect processes tunnel/redirect response
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

// handlePicker processes picker response (multiple files)
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

// downloadFile downloads a file from direct URL
func downloadFile(mediaURL, suggestedFilename string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "aether-scrape-")
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", mediaURL, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := downloadClient.Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
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
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("file too small: %d bytes", size)
	}

	return filePath, nil
}

// determineFileExtension extracts file extension from various sources
func determineFileExtension(resp *http.Response, mediaURL, suggestedFilename string) string {
	// Priority 1: Suggested filename
	if ext := filepath.Ext(suggestedFilename); ext != "" {
		return ext
	}

	// Priority 2: Content-Type header
	contentType := resp.Header.Get("Content-Type")
	for ct, ext := range contentTypeToExt {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}

	// Priority 3: URL path
	if parsedURL, err := url.Parse(mediaURL); err == nil {
		if ext := filepath.Ext(parsedURL.Path); ext != "" {
			return ext
		}
	}

	return ".tmp"
}

// DownloadImage downloads an image file
func DownloadImage(imageURL string) (string, int64, error) {
	tmpDir, err := os.MkdirTemp("", "aether-image-")
	if err != nil {
		return "", 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cobaltTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(tmpDir)
		return "", 0, fmt.Errorf("HTTP %d", resp.StatusCode)
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
		os.RemoveAll(tmpDir)
		return "", 0, fmt.Errorf("image too small: %d bytes", size)
	}

	return filePath, size, nil
}

// getImageExtension maps content type to image extension
func getImageExtension(contentType string) string {
	for ct, ext := range imageContentTypes {
		if strings.Contains(contentType, ct) {
			return ext
		}
	}
	return ".jpg"
}
