package downloader

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type DownloadMetrics struct {
	StartTime  time.Time
	EndTime    time.Time
	FileSize   int64
	FilesCount int
	Provider   string
	Duration   time.Duration
	AvgSpeed   float64
}

func (m *DownloadMetrics) LogMetrics() {
	m.Duration = m.EndTime.Sub(m.StartTime)
	if m.Duration.Seconds() > 0 {
		m.AvgSpeed = float64(m.FileSize/(1024*1024)) / m.Duration.Seconds()
	}

	log.Printf(
		" Download Metrics: %d files, %.2f MB, %.2f MB/s, %s provider, %.1fs duration",
		m.FilesCount, float64(m.FileSize)/(1024*1024), m.AvgSpeed, m.Provider, m.Duration.Seconds(),
	)
}

func DownloadVideo(url string) ([]string, int64, string, error) {
	return runYTDLP(url, false)
}

func DownloadVideoWithProgress(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, int64, string, error) {
	return runYTDLPWithProgress(url, false, bot, chatID, msgID)
}

func GetVideoInfo(url string) (int64, error) {
	if !isYouTubeURL(url) {
		return 0, fmt.Errorf("unsupported platform")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := []string{
		"--no-download",
		"--get-url",
		"--get-format",
		"--no-playlist",
		url,
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	_, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get video info: %w", err)
	}

	return 0, nil
}

func DownloadAudio(url string) ([]string, int64, string, error) {
	return runYTDLP(url, true)
}

func DownloadAudioWithProgress(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, int64, string, error) {
	if filePaths, err := DownloadMediaWithCobalt(url, true); err == nil {
		return calculateTotalSize(filePaths, "Cobalt")
	}

	if isTikTokURL(url) {
		log.Printf("Using TikTok API for audio download (no progress)")
		return DownloadTikTokAudioWithProgress(url, bot, chatID, msgID)
	}

	if isYouTubeURL(url) {
		log.Printf("Using yt-dlp for YouTube audio download (no progress)")
		return runYTDLP(url, true)
	}

	return nil, 0, "", fmt.Errorf("unsupported platform or download failed")
}

func runYTDLP(url string, audioOnly bool) ([]string, int64, string, error) {
	if filePaths, err := DownloadMediaWithCobalt(url, audioOnly); err == nil {
		return calculateTotalSize(filePaths, "Cobalt")
	}

	if audioOnly && isTikTokURL(url) {
		log.Printf("Using TikTok API for audio download")
		filePath, _, _, err := DownloadTikTokAudio(url)
		if err != nil {
			return nil, 0, "", fmt.Errorf("TikTok audio download failed: %w", err)
		}
		return calculateTotalSize([]string{filePath}, "TikTok (tikwm API)")
	}

	if !isYouTubeURL(url) {
		return nil, 0, "", fmt.Errorf("unsupported platform or download failed")
	}

	log.Printf("Falling back to yt-dlp for YouTube URL")

	filePaths, err := DownloadMediaWithYTDLP(url, audioOnly, false)
	if err != nil {
		log.Printf("yt-dlp without cookies failed, retrying with cookies...")
		filePaths, err = DownloadMediaWithYTDLP(url, audioOnly, true)
		if err != nil {
			return nil, 0, "", fmt.Errorf("all download methods failed: %w", err)
		}
	}

	provider := "yt-dlp (multithreaded)"
	return calculateTotalSize(filePaths, provider)
}

func runYTDLPWithProgress(url string, audioOnly bool, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, int64, string, error) {
	if filePaths, err := DownloadMediaWithCobalt(url, audioOnly); err == nil {
		return calculateTotalSize(filePaths, "Cobalt")
	}

	if audioOnly && isTikTokURL(url) {
		log.Printf("Using TikTok API for audio download")
		filePath, _, _, err := DownloadTikTokAudio(url)
		if err != nil {
			return nil, 0, "", fmt.Errorf("TikTok audio download failed: %w", err)
		}
		return calculateTotalSize([]string{filePath}, "TikTok (tikwm API)")
	}

	if !isYouTubeURL(url) {
		return nil, 0, "", fmt.Errorf("unsupported platform or download failed")
	}

	log.Printf("Falling back to yt-dlp for YouTube URL with progress")

	filePaths, err := DownloadMediaWithYTDLPWithProgress(url, audioOnly, false, bot, chatID, msgID)
	if err != nil {
		log.Printf("yt-dlp without cookies failed, retrying with cookies...")
		filePaths, err = DownloadMediaWithYTDLPWithProgress(url, audioOnly, true, bot, chatID, msgID)
		if err != nil {
			return nil, 0, "", fmt.Errorf("all download methods failed: %w", err)
		}
	}

	provider := "yt-dlp (multithreaded)"
	return calculateTotalSize(filePaths, provider)
}

func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func isTikTokURL(url string) bool {
	return strings.Contains(url, "tiktok.com")
}

func getCookiePath() string {
	if path := os.Getenv("YTDLP_COOKIES"); path != "" {
		return path
	}
	return "cookies.txt"
}

func DownloadMediaWithYTDLPWithProgress(mediaURL string, audioOnly, useCookies bool, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, error) {
	metrics := &DownloadMetrics{
		StartTime: time.Now(),
		Provider:  "yt-dlp",
	}

	log.Printf(" Starting download: url=%s, audio=%v, cookies=%v",
		mediaURL, audioOnly, useCookies)

	tmpDir, err := os.MkdirTemp("", "aether-ytdlp-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpTimeout)
	defer cancel()

	if bot != nil && GetCPUManager().IsEnabled() {
		go GetCPUManager().MonitorCPUDuringDownload(ctx, bot, chatID, msgID)
	}

	args := buildYTDLPArgs(tmpDir, audioOnly, useCookies)
	args = append(args, "--newline", "--progress", mediaURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

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

	go trackYTDLPProgress(stdout, bot, chatID, msgID)

	if err := cmd.Wait(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	if len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no files downloaded from %s", mediaURL)
	}

	var validFiles []string
	for _, f := range files {
		if !strings.HasPrefix(filepath.Base(f), ".") {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no valid files found in download output")
	}

	for _, f := range validFiles {
		if info, err := os.Stat(f); err == nil {
			metrics.FileSize += info.Size()
		}
	}
	metrics.FilesCount = len(validFiles)
	metrics.EndTime = time.Now()
	metrics.LogMetrics()

	log.Printf(" Downloaded %d file(s) to %s", len(validFiles), tmpDir)

	return validFiles, nil
}

func trackYTDLPProgress(stdout io.Reader, bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	trackYTDLPProgressWithFilename(stdout, bot, chatID, msgID, "", "")
}

func trackYTDLPProgressWithFilename(stdout io.Reader, bot *tgbotapi.BotAPI, chatID int64, msgID int, fileName string, totalSize string) {
	scanner := bufio.NewScanner(stdout)
	lastUpdate := time.Now()
	updateInterval := 4 * time.Second
	updateTimeout := 10 * time.Second
	platform := "YouTube"

	log.Printf(" [Progress] Starting progress tracker")

	if fileName != "" {
		UpdateInitialProgressMessageDetailed(bot, chatID, msgID, fileName, totalSize, platform, "user")
	} else {
		UpdateInitialProgressMessage(bot, chatID, msgID, platform)
	}

	lineCount := 0
	var lastPercentage float64

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount%10 == 0 {
			log.Printf(" [Progress] Processed %d lines", lineCount)
		}

		if matches := ytdlpProgressRegex.FindStringSubmatch(line); len(matches) > 0 {
			if time.Since(lastUpdate) < updateInterval {
				continue
			}

			percentage, _ := strconv.ParseFloat(matches[1], 64)
			size := matches[2]
			var speed, eta string

			if matches[3] != "" && matches[4] != "" {
				eta = matches[3]
				speed = matches[4]
			} else if matches[5] != "" && matches[6] != "" {
				speed = matches[5]
				eta = matches[6]
			} else {
				continue
			}

			if percentage == lastPercentage && percentage < 100 {
				continue
			}

			progress := DownloadProgress{
				Percentage: percentage,
				Downloaded: size,
				Speed:      speed,
				ETA:        eta,
				Status:     "Downloading",
			}

			if fileName != "" {
				UpdateProgressMessageDetailed(bot, chatID, msgID, fileName, progress, totalSize, platform, "user")
			} else {
				UpdateProgressMessage(bot, chatID, msgID, platform, progress)
			}

			lastPercentage = percentage
			lastUpdate = time.Now()
		} else if strings.Contains(line, "[download]") {
			if time.Since(lastUpdate) > updateTimeout {
				log.Printf(" [Progress] Status: %s", line)
				lastUpdate = time.Now()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf(" [Progress] ❌ Scanner error: %v", err)
	}

	log.Printf(" [Progress] ✅ Finished (processed %d lines)", lineCount)
}

func DownloadMediaWithYTDLP(mediaURL string, audioOnly, useCookies bool) ([]string, error) {

	return DownloadMediaWithYTDLPWithProgress(mediaURL, audioOnly, useCookies, nil, 0, 0)
}

func buildYTDLPArgs(tmpDir string, audioOnly, useCookies bool) []string {
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
		"--abort-on-error",
		"--no-warnings",
		"-N", "16",
	}

	log.Printf("Using yt-dlp with 16 concurrent threads")

	if useCookies {
		cookiePath := getCookiePath()
		if _, err := os.Stat(cookiePath); err == nil {
			args = append(args, "--cookies", cookiePath)
			log.Printf(" Using cookies from: %s", cookiePath)
		}
	}

	return args
}

func DownloadVideoWithProgressDetailed(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string) ([]string, int64, string, error) {
	return runYTDLPWithProgressDetailed(url, false, bot, chatID, msgID, username)
}

func runYTDLPWithProgressDetailed(url string, audioOnly bool, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string) ([]string, int64, string, error) {
	if filePaths, err := DownloadMediaWithCobalt(url, audioOnly); err == nil {
		return calculateTotalSize(filePaths, "Cobalt")
	}

	if audioOnly && isTikTokURL(url) {
		log.Printf("Using TikTok API for audio download")
		filePath, _, _, err := DownloadTikTokAudio(url)
		if err != nil {
			return nil, 0, "", fmt.Errorf("TikTok audio download failed: %w", err)
		}
		return calculateTotalSize([]string{filePath}, "TikTok (tikwm API)")
	}

	if !isYouTubeURL(url) {
		return nil, 0, "", fmt.Errorf("unsupported platform or download failed")
	}

	log.Printf("Falling back to yt-dlp for YouTube URL with progress")

	filePaths, err := DownloadMediaWithYTDLPWithProgressDetailed(url, audioOnly, false, bot, chatID, msgID, username)
	if err != nil {
		log.Printf("yt-dlp without cookies failed, retrying with cookies...")
		filePaths, err = DownloadMediaWithYTDLPWithProgressDetailed(url, audioOnly, true, bot, chatID, msgID, username)
		if err != nil {
			return nil, 0, "", fmt.Errorf("all download methods failed: %w", err)
		}
	}

	provider := "yt-dlp (multithreaded)"
	return calculateTotalSize(filePaths, provider)
}

func DownloadMediaWithYTDLPWithProgressDetailed(mediaURL string, audioOnly, useCookies bool, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string) ([]string, error) {
	metrics := &DownloadMetrics{
		StartTime: time.Now(),
		Provider:  "yt-dlp",
	}

	log.Printf(" Starting download: url=%s, audio=%v, cookies=%v",
		mediaURL, audioOnly, useCookies)

	tmpDir, err := os.MkdirTemp("", "aether-ytdlp-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ytdlpTimeout)
	defer cancel()

	if bot != nil && GetCPUManager().IsEnabled() {
		go GetCPUManager().MonitorCPUDuringDownload(ctx, bot, chatID, msgID)
	}

	args := buildYTDLPArgs(tmpDir, audioOnly, useCookies)
	args = append(args, "--newline", "--progress", mediaURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

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

	go trackYTDLPProgressDetailed(stdout, bot, chatID, msgID, username, tmpDir)

	if err := cmd.Wait(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to glob files: %w", err)
	}

	if len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no files downloaded from %s", mediaURL)
	}

	var validFiles []string
	for _, f := range files {
		if !strings.HasPrefix(filepath.Base(f), ".") {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no valid files found in download output")
	}

	for _, f := range validFiles {
		if info, err := os.Stat(f); err == nil {
			metrics.FileSize += info.Size()
		}
	}
	metrics.FilesCount = len(validFiles)
	metrics.EndTime = time.Now()
	metrics.LogMetrics()

	log.Printf(" Downloaded %d file(s) to %s", len(validFiles), tmpDir)

	return validFiles, nil
}

func trackYTDLPProgressDetailed(stdout io.Reader, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string, tmpDir string) {
	scanner := bufio.NewScanner(stdout)
	lastUpdate := time.Now()
	updateInterval := 4 * time.Second
	updateTimeout := 10 * time.Second
	platform := "YouTube"

	log.Printf(" [Progress] Starting detailed progress tracker")

	UpdateInitialProgressMessageDetailed(bot, chatID, msgID, "video.mp4", "--", platform, username)

	lineCount := 0
	var lastPercentage float64

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount%10 == 0 {
			log.Printf(" [Progress] Processed %d lines", lineCount)
		}

		if matches := ytdlpProgressRegex.FindStringSubmatch(line); len(matches) > 0 {
			if time.Since(lastUpdate) < updateInterval {
				continue
			}

			percentage, _ := strconv.ParseFloat(matches[1], 64)
			size := matches[2]
			var speed, eta string

			if matches[3] != "" && matches[4] != "" {
				eta = matches[3]
				speed = matches[4]
			} else if matches[5] != "" && matches[6] != "" {
				speed = matches[5]
				eta = matches[6]
			} else {
				continue
			}

			if percentage == lastPercentage && percentage < 100 {
				continue
			}

			progress := DownloadProgress{
				Percentage: percentage,
				Downloaded: size,
				Speed:      speed,
				ETA:        eta,
				Status:     "Downloading",
			}

			fileName := "video.mp4"
			files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
			if err == nil && len(files) > 0 {
				for _, f := range files {
					if !strings.HasPrefix(filepath.Base(f), ".") {
						fileName = filepath.Base(f)
						break
					}
				}
			}

			UpdateProgressMessageDetailed(bot, chatID, msgID, fileName, progress, size, platform, username)

			lastPercentage = percentage
			lastUpdate = time.Now()
		} else if strings.Contains(line, "[download]") {
			if time.Since(lastUpdate) > updateTimeout {
				log.Printf(" [Progress] Status: %s", line)
				lastUpdate = time.Now()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf(" [Progress] ❌ Scanner error: %v", err)
	}

	log.Printf(" [Progress] ✅ Finished (processed %d lines)", lineCount)
}

func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)

	if strings.HasSuffix(sizeStr, "MiB") {
		sizeStr = strings.TrimSuffix(sizeStr, "MiB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024)
		}
	} else if strings.HasSuffix(sizeStr, "GiB") {
		sizeStr = strings.TrimSuffix(sizeStr, "GiB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024 * 1024)
		}
	} else if strings.HasSuffix(sizeStr, "KiB") {
		sizeStr = strings.TrimSuffix(sizeStr, "KiB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024)
		}
	} else if strings.HasSuffix(sizeStr, "MB") {
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024)
		}
	} else if strings.HasSuffix(sizeStr, "GB") {
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024 * 1024)
		}
	} else if strings.HasSuffix(sizeStr, "KB") {
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024)
		}
	} else if strings.HasSuffix(sizeStr, "B") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			return size
		}
	}

	return 0
}
