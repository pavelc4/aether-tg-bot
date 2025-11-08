package providers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/core"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/ui"
)

const (
	youtubeTimeout = 10 * time.Minute
	KB             = 1024
	MB             = KB * 1024
	GB             = MB * 1024
	TB             = GB * 1024
)

type YouTubeProvider struct {
	timeout    time.Duration
	useCookies bool
	bot        *tgbotapi.BotAPI
	chatID     int64
	msgID      int
	username   string
	fileName   string
	totalSize  string
}

func NewYouTubeProvider(useCookies bool) *YouTubeProvider {
	return &YouTubeProvider{
		timeout:    youtubeTimeout,
		useCookies: useCookies,
	}
}

func NewYouTubeProviderWithProgress(useCookies bool, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string) *YouTubeProvider {
	return &YouTubeProvider{
		timeout:    youtubeTimeout,
		useCookies: useCookies,
		bot:        bot,
		chatID:     chatID,
		msgID:      msgID,
		username:   username,
		fileName:   "YouTube Video",
		totalSize:  "Unknown",
	}
}

func (yp *YouTubeProvider) Name() string {
	return "YouTube (yt-dlp)"
}

func (yp *YouTubeProvider) CanHandle(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func (yp *YouTubeProvider) Download(ctx context.Context, url string, audioOnly bool) ([]string, error) {
	log.Printf("YouTube: Starting download (audio=%v, cookies=%v)", audioOnly, yp.useCookies)

	tmpDir, err := os.MkdirTemp("", "aether-youtube-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory failed: %w", err)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), yp.timeout)
		defer cancel()
	}

	args := yp.buildArgs(tmpDir, audioOnly)
	args = append(args, url)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	// Setup progress tracking if bot is available
	if yp.bot != nil && yp.chatID != 0 && yp.msgID != 0 {
		// Send initial progress message
		ui.UpdateInitialProgressMessageDetailed(
			yp.bot, yp.chatID, yp.msgID,
			yp.fileName, yp.totalSize,
			"YouTube", yp.username,
		)

		// ✅ Use StdoutPipe (NOT StderrPipe!) because yt-dlp writes progress to stdout
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		// ✅ Start tracking progress in goroutine
		go yp.trackProgress(stdoutPipe)

		// Stderr for error messages only
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		log.Printf("YouTube: Executing yt-dlp with %d args (with progress tracking)", len(args))

		// ✅ Use Start() instead of Run() to allow goroutine to work
		if err := cmd.Start(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
		}

		// ✅ Wait for command to complete
		if err := cmd.Wait(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
		}
	} else {
		// No progress tracking, just run command
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		log.Printf("YouTube: Executing yt-dlp with %d args (no progress)", len(args))

		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
		}
	}

	// Collect downloaded files
	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("list files failed: %w", err)
	}

	if len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no files downloaded")
	}

	var validFiles []string
	for _, f := range files {
		if !strings.HasPrefix(filepath.Base(f), ".") {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no valid files found")
	}

	log.Printf("YouTube: Downloaded %d file(s)", len(validFiles))
	return validFiles, nil
}

func (yp *YouTubeProvider) trackProgress(stdoutPipe io.ReadCloser) {
	defer stdoutPipe.Close()

	scanner := bufio.NewScanner(stdoutPipe)
	// Increase buffer for large files
	buf := make([]byte, 256*1024) // 256KB buffer
	scanner.Buffer(buf, 256*1024)

	completeRegex := regexp.MustCompile(`\[download\]\s+100%\s+of\s+~?([\d.]+)([KMGT]iB)`)
	lastUpdate := time.Now()
	var mu sync.Mutex
	var allFileSizes []float64
	var currentFileSize float64 = 0

	// Dynamic update interval based on file size
	minTelegramInterval := 1500 * time.Millisecond // Prevent Telegram rate limit

	var lastPercentage float64 = -1
	lineCount := 0

	log.Printf("YouTube: Starting progress tracker")

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		// Log every 10 lines for debugging (optional, remove after testing)
		if lineCount%10 == 0 {
			log.Printf("YouTube: Processed %d lines", lineCount)
		}

		// Check for destination line
		if strings.Contains(line, "[download] Destination:") {
			currentFileSize = 0
			continue
		}

		// Handle 100% completion
		completeMatches := completeRegex.FindStringSubmatch(line)
		if len(completeMatches) > 1 {
			size, _ := strconv.ParseFloat(completeMatches[1], 64)
			unit := completeMatches[2]
			fileSize := convertToBytes(size, unit)
			allFileSizes = append(allFileSizes, fileSize)
			currentFileSize = fileSize
			log.Printf("YouTube: File completed - %.2f MB", fileSize/MB)

			totalSizeBytes := float64(0)
			for _, fs := range allFileSizes {
				totalSizeBytes += fs
			}

			mu.Lock()
			progress := ui.DownloadProgress{
				Percentage: 100.0,
				Downloaded: formatBytes(totalSizeBytes),
				Speed:      "Complete",
				ETA:        "Done",
				Status:     "Completed",
			}

			ui.UpdateProgressMessageDetailed(
				yp.bot, yp.chatID, yp.msgID,
				yp.fileName,
				progress,
				formatBytes(totalSizeBytes),
				"YouTube",
				yp.username,
			)

			lastUpdate = time.Now()
			mu.Unlock()
			continue
		}

		// Parse progress with regex
		matches := core.YTDLPProgressRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			percentage, _ := strconv.ParseFloat(matches[1], 64)
			totalSizeStr := matches[2]
			totalSizeParsed, unit := parseSize(totalSizeStr)
			currentFileSize = convertToBytes(totalSizeParsed, unit)

			var speedStr, eta string
			if matches[3] != "" && matches[4] != "" {
				speedStr = matches[4]
				eta = matches[3]
			} else if matches[5] != "" && matches[6] != "" {
				speedStr = matches[5]
				eta = matches[6]
			}

			var speedFormatted string
			if speedStr != "" {
				speedParsed, speedUnit := parseSize(speedStr)
				speedBytes := convertToBytes(speedParsed, speedUnit)
				speedFormatted = formatBytes(speedBytes) + "/s"
			} else {
				speedFormatted = "--"
			}

			// Calculate overall progress
			totalSizeBytes := float64(0)
			for _, fileSize := range allFileSizes {
				totalSizeBytes += fileSize
			}

			totalSizeBytes += currentFileSize
			downloadedCurrentFile := (percentage / 100) * currentFileSize
			totalDownloadedBytes := downloadedCurrentFile
			for _, fileSize := range allFileSizes {
				totalDownloadedBytes += fileSize
			}

			overallPercentage := (totalDownloadedBytes / totalSizeBytes) * 100
			if totalSizeBytes == 0 {
				overallPercentage = percentage
			}

			totalSizeFormatted := formatBytes(totalSizeBytes)
			downloadedFormatted := formatBytes(totalDownloadedBytes)

			mu.Lock()
			now := time.Now()
			timeSinceLastUpdate := now.Sub(lastUpdate)

			// Adaptive update strategy for large files
			shouldUpdate := false

			// Always enforce minimum Telegram interval
			if timeSinceLastUpdate < minTelegramInterval {
				mu.Unlock()
				continue
			}

			// For files > 25MB, use larger intervals
			if currentFileSize > 25*MB {
				// Update every 3 seconds for large files
				shouldUpdate = timeSinceLastUpdate >= 3*time.Second
				// OR if percentage changed by at least 5%
				shouldUpdate = shouldUpdate || (math.Abs(percentage-lastPercentage) >= 5.0)
			} else if currentFileSize > 10*MB {
				// Medium files: every 2 seconds or 3% change
				shouldUpdate = timeSinceLastUpdate >= 2*time.Second
				shouldUpdate = shouldUpdate || (math.Abs(percentage-lastPercentage) >= 3.0)
			} else {
				// Small files: every 1.5 seconds or percentage change
				shouldUpdate = timeSinceLastUpdate >= minTelegramInterval
				shouldUpdate = shouldUpdate || (percentage != lastPercentage)
			}

			if shouldUpdate {
				progress := ui.DownloadProgress{
					Percentage: overallPercentage,
					Downloaded: downloadedFormatted,
					Speed:      speedFormatted,
					ETA:        eta,
					Status:     "Downloading",
				}

				ui.UpdateProgressMessageDetailed(
					yp.bot, yp.chatID, yp.msgID,
					yp.fileName,
					progress,
					totalSizeFormatted,
					"YouTube",
					yp.username,
				)

				lastPercentage = percentage
				lastUpdate = now
				log.Printf("YouTube: Progress updated - %.1f%% (%.2f MB)", overallPercentage, currentFileSize/MB)
			}

			mu.Unlock()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("YouTube: Scanner error: %v", err)
	}

	// Final update at 100%
	mu.Lock()
	finalSize := ""
	finalPercentage := 100.0

	if len(allFileSizes) > 0 {
		finalTotalBytes := float64(0)
		for _, fileSize := range allFileSizes {
			finalTotalBytes += fileSize
		}
		finalSize = formatBytes(finalTotalBytes)
	} else if currentFileSize > 0 {
		finalSize = formatBytes(currentFileSize)
	} else {
		finalSize = yp.totalSize
	}

	progress := ui.DownloadProgress{
		Percentage: finalPercentage,
		Downloaded: finalSize,
		Speed:      "Complete",
		ETA:        "Done",
		Status:     "Completed",
	}

	ui.UpdateProgressMessageDetailed(
		yp.bot, yp.chatID, yp.msgID,
		yp.fileName,
		progress,
		finalSize,
		"YouTube",
		yp.username,
	)

	mu.Unlock()

	log.Printf("YouTube: Progress tracking finished (processed %d lines)", lineCount)
}

func (yp *YouTubeProvider) buildArgs(tmpDir string, audioOnly bool) []string {
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
		"--newline",  // Force newline after each progress update
		"--progress", // Enable progress output
	}

	log.Printf("YouTube: Using yt-dlp with 16 concurrent threads")

	if yp.useCookies {
		cookiePath := yp.getCookiePath()
		if _, err := os.Stat(cookiePath); err == nil {
			args = append(args, "--cookies", cookiePath)
			log.Printf("YouTube: Using cookies from: %s", cookiePath)
		}
	}

	return args
}

func (yp *YouTubeProvider) getCookiePath() string {
	if path := os.Getenv("YTDLP_COOKIES"); path != "" {
		return path
	}
	return "cookies.txt"
}

func convertToBytes(size float64, unit string) float64 {
	switch unit {
	case "KiB", "KB":
		return size * KB
	case "MiB", "MB":
		return size * MB
	case "GiB", "GB":
		return size * GB
	case "TiB", "TB":
		return size * TB
	case "KiB/s", "KB/s":
		return size * KB
	case "MiB/s", "MB/s":
		return size * MB
	case "GiB/s", "GB/s":
		return size * GB
	case "TiB/s", "TB/s":
		return size * TB
	default:
		return size
	}
}

func formatBytes(bytes float64) string {
	if bytes < KB {
		return fmt.Sprintf("%.2f B", bytes)
	} else if bytes < MB {
		return fmt.Sprintf("%.2f KB", bytes/KB)
	} else if bytes < GB {
		return fmt.Sprintf("%.2f MB", bytes/MB)
	} else if bytes < TB {
		return fmt.Sprintf("%.2f GB", bytes/GB)
	} else {
		return fmt.Sprintf("%.2f TB", bytes/TB)
	}
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseSize(sizeStr string) (float64, string) {
	re := regexp.MustCompile(`([\d.]+)([A-Za-z]+)`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) >= 3 {
		return parseFloat(matches[1]), matches[2]
	}
	return parseFloat(sizeStr), ""
}
