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

	if yp.bot != nil && yp.chatID != 0 && yp.msgID != 0 {
		ui.UpdateInitialProgressMessageDetailed(
			yp.bot, yp.chatID, yp.msgID,
			yp.fileName, yp.totalSize,
			"YouTube", yp.username,
		)

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		go yp.trackProgress(stdoutPipe)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		log.Printf("YouTube: Executing yt-dlp with %d args (with progress tracking)", len(args))

		if err := cmd.Start(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
		}
	} else {
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		log.Printf("YouTube: Executing yt-dlp with %d args (no progress)", len(args))

		if err := cmd.Run(); err != nil {
			os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
		}
	}

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
	buf := make([]byte, 256*1024)
	scanner.Buffer(buf, 256*1024)

	completeRegex := regexp.MustCompile(`\[download\]\s+100%\s+of\s+~?([\d.]+)([KMGT]iB)`)
	lastUpdate := time.Now()
	var mu sync.Mutex
	var allFileSizes []float64
	var currentFileSize float64 = 0

	minTelegramInterval := 1500 * time.Millisecond
	var lastPercentage float64 = -1
	lineCount := 0

	log.Printf("YouTube: Starting progress tracker")

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount%10 == 0 {
			log.Printf("YouTube: Processed %d lines", lineCount)
		}

		if strings.Contains(line, "[download] Destination:") {
			currentFileSize = 0
			continue
		}

		completeMatches := completeRegex.FindStringSubmatch(line)
		if len(completeMatches) > 1 {
			size, _ := strconv.ParseFloat(completeMatches[1], 64)
			unit := completeMatches[2]
			fileSize := core.ConvertToBytes(size, unit)
			allFileSizes = append(allFileSizes, fileSize)
			currentFileSize = fileSize
			log.Printf("YouTube: File completed - %.2f MB", fileSize/core.MB)

			totalSizeBytes := float64(0)
			for _, fs := range allFileSizes {
				totalSizeBytes += fs
			}

			mu.Lock()
			progress := ui.DownloadProgress{
				Percentage: 100.0,
				Downloaded: core.FormatBytes(totalSizeBytes),
				Speed:      "Complete",
				ETA:        "Done",
				Status:     "Completed",
			}

			ui.UpdateProgressMessageDetailed(
				yp.bot, yp.chatID, yp.msgID,
				yp.fileName,
				progress,
				core.FormatBytes(totalSizeBytes),
				"YouTube",
				yp.username,
			)

			lastUpdate = time.Now()
			mu.Unlock()
			continue
		}

		matches := core.YTDLPProgressRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			percentage, _ := strconv.ParseFloat(matches[1], 64)
			totalSizeStr := matches[2]
			totalSizeParsed, unit := parseSize(totalSizeStr)
			currentFileSize = core.ConvertToBytes(totalSizeParsed, unit)

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
				speedBytes := core.ConvertToBytes(speedParsed, speedUnit)
				speedFormatted = core.FormatBytes(speedBytes) + "/s"
			} else {
				speedFormatted = "--"
			}

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

			totalSizeFormatted := core.FormatBytes(totalSizeBytes)
			downloadedFormatted := core.FormatBytes(totalDownloadedBytes)

			mu.Lock()
			now := time.Now()
			timeSinceLastUpdate := now.Sub(lastUpdate)

			shouldUpdate := false

			if timeSinceLastUpdate < minTelegramInterval {
				mu.Unlock()
				continue
			}

			if currentFileSize > 25*core.MB {
				shouldUpdate = timeSinceLastUpdate >= 3*time.Second
				shouldUpdate = shouldUpdate || (math.Abs(percentage-lastPercentage) >= 5.0)
			} else if currentFileSize > 10*core.MB {
				shouldUpdate = timeSinceLastUpdate >= 2*time.Second
				shouldUpdate = shouldUpdate || (math.Abs(percentage-lastPercentage) >= 3.0)
			} else {
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
				log.Printf("YouTube: Progress updated - %.1f%% (%.2f MB)", overallPercentage, currentFileSize/core.MB)
			}

			mu.Unlock()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("YouTube: Scanner error: %v", err)
	}

	mu.Lock()
	finalSize := ""
	finalPercentage := 100.0

	if len(allFileSizes) > 0 {
		finalTotalBytes := float64(0)
		for _, fileSize := range allFileSizes {
			finalTotalBytes += fileSize
		}
		finalSize = core.FormatBytes(finalTotalBytes)
	} else if currentFileSize > 0 {
		finalSize = core.FormatBytes(currentFileSize)
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
		"--newline",
		"--progress",
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
