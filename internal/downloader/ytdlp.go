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

// DownloadVideo downloads video using yt-dlp with Cobalt fallback
func DownloadVideo(url string) ([]string, int64, string, error) {
	return runYTDLP(url, false)
}

// DownloadAudio downloads audio using yt-dlp with Cobalt fallback
func DownloadAudio(url string) ([]string, int64, string, error) {
	return runYTDLP(url, true)
}

// runYTDLP orchestrates download with Cobalt first, then yt-dlp fallback
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

// isYouTubeURL checks if URL is from YouTube
func isYouTubeURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
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

// DownloadMediaWithYTDLP is a wrapper untuk backward compatibility
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

// isAria2Available checks if aria2c is installed
func isAria2Available() bool {
	_, err := exec.LookPath("aria2c")
	return err == nil
}
