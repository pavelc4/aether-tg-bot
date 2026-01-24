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
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func (yp *YouTubeProvider) GetVideoInfo(ctx context.Context, url string) (*VideoInfo, error) {
	args := []string{
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
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

	filename := fmt.Sprintf("%s.%s", meta.Title, meta.Ext)
	filename = strings.ReplaceAll(filename, "/", "_")

	return &VideoInfo{
		URL:      meta.URL, // Direct URL
		FileName: filename,
		FileSize: meta.FileSize, // Might be 0 if unknown
		MimeType: "video/" + meta.Ext,
		Duration: int(meta.Duration),
		Headers:  meta.HttpHeaders,
	}, nil
}

type ytdlpMeta struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	URL         string            `json:"url"`
	Ext         string            `json:"ext"`
	Duration    float64           `json:"duration"`
	FileSize    int64             `json:"filesize,omitempty"`     // Sometimes separate
	FileSizeApp int64             `json:"filesize_approx,omitempty"` // Fallback
	HttpHeaders map[string]string `json:"http_headers"`
}
