package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

const youtubeTimeout = 10 * time.Minute

type YouTubeProvider struct {
}

func NewYouTube() *YouTubeProvider {
	return &YouTubeProvider{}
}

func (yp *YouTubeProvider) Name() string {
	return "YouTube"
}

func (yp *YouTubeProvider) Supports(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") || strings.Contains(url, "tiktok.com")
}

func (yp *YouTubeProvider) GetVideoInfo(ctx context.Context, url string) (*VideoInfo, error) {
	args := []string{
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
		"-f", "best[ext=mp4]/best", // Force progressive format for single-stream retrieval
		url,
	}

	if cookies := config.GetYtdlpCookies(); cookies != "" {
		if _, err := os.Stat(cookies); err == nil {
			args = append(args, "--cookies", cookies)
		}
	}

	ctx, cancel := context.WithTimeout(ctx, youtubeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp failed: %w (stderr: %s)", err, stderr.String())
	}

	var meta ytdlpMeta
	if err := json.Unmarshal(stdout.Bytes(), &meta); err != nil {
		return nil, fmt.Errorf("decode json failed: %w", err)
	}

	finalURL := meta.URL
	if (finalURL == "" || isImageURL(finalURL)) && len(meta.Formats) > 0 {
		// Find a valid video URL in formats
		for _, f := range meta.Formats {
			if f.URL != "" && f.VCodec != "none" && !isImageURL(f.URL) {
				finalURL = f.URL
				break
			}
		}
	}

	if finalURL == "" || isImageURL(finalURL) {
		logger.Error("yt-dlp returned no valid video URL", "url", finalURL, "stderr", stderr.String())
		return nil, fmt.Errorf("no streamable video URL found")
	}

	filename := fmt.Sprintf("%s.%s", meta.Title, meta.Ext)
	filename = strings.ReplaceAll(filename, "/", "_")

	return &VideoInfo{
		URL:      finalURL,
		FileName: filename,
		FileSize: meta.FileSize,
		MimeType: "video/" + meta.Ext,
		Duration: int(meta.Duration),
		Headers:  meta.HttpHeaders,
	}, nil
}

func isImageURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".png") || strings.Contains(lower, ".webp")
}

type ytdlpMeta struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	URL         string            `json:"url"`
	Ext         string            `json:"ext"`
	Duration    float64           `json:"duration"`
	FileSize    int64             `json:"filesize,omitempty"`
	FileSizeApp int64             `json:"filesize_approx,omitempty"`
	HttpHeaders map[string]string `json:"http_headers"`
	Formats     []ytdlpFormat     `json:"formats"`
}

type ytdlpFormat struct {
	URL    string `json:"url"`
	VCodec string `json:"vcodec"`
}
