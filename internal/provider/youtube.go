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

func (yp *YouTubeProvider) GetVideoInfo(ctx context.Context, url string, opts Options) ([]VideoInfo, error) {
	formatArg := "bestvideo[height<=1080]+bestaudio/best[height<=1080]/bestvideo+bestaudio/best"
	if opts.AudioOnly {
		formatArg = "bestaudio[ext=m4a]/bestaudio"
	}

	args := []string{
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
		"--rm-cache-dir",
		"--js-runtimes", "bun",
		"--retries", "10",
		"--fragment-retries", "10",
		"-f", formatArg,
		url,
	}

	if cookies := config.GetYtdlpCookies(); cookies != "" {
		if _, err := os.Stat(cookies); err == nil {
			logger.Info("Using yt-dlp cookies", "path", cookies)
			args = append(args, "--cookies", cookies)
		} else {
			logger.Warn("Cookies file not found", "path", cookies)
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

	usePipe := true
	finalURL := url 
	
	filename := fmt.Sprintf("%s.%s", meta.Title, meta.Ext)
	filename = strings.ReplaceAll(filename, "/", "_")
	if !opts.AudioOnly && meta.Ext != "mp4" {
		filename = strings.Replace(filename, "."+meta.Ext, ".mp4", 1)
		meta.Ext = "mp4"
	}

	size := meta.FileSize
	if size == 0 {
		size = meta.FileSizeApp
	}

	if size == 0 && meta.TBR > 0 && meta.Duration > 0 {
		logger.Info("Estimating size from bitrate", "tbr", meta.TBR, "duration", meta.Duration)
		size = int64((meta.TBR * 1000 * meta.Duration) / 8)
	}

	logger.Info("YouTube info resolved", 
		"title", meta.Title, 
		"size", size,
		"res", fmt.Sprintf("%dx%d", meta.Width, meta.Height),
		"dur", meta.Duration,
		"pipe", usePipe,
	)
	
	mime := "video/mp4"
	if opts.AudioOnly || meta.Ext == "m4a" || meta.Ext == "mp3" {
		mime = "audio/" + meta.Ext
		if meta.Ext == "m4a" {
			mime = "audio/mp4"
		}
	}

	return []VideoInfo{{
		URL:      finalURL,
		FileName: filename,
		Title:    meta.Title,
		FileSize: size,
		MimeType: mime,
		Duration: int(meta.Duration),
		Width:    meta.Width,
		Height:   meta.Height,
		Headers:  meta.HttpHeaders,
		UsePipe:  usePipe,
	}}, nil
}

func isNonStreamableURL(url string) bool {
	lower := strings.ToLower(url)
	imgExts := []string{".jpg", ".jpeg", ".png", ".webp"}
	for _, ext := range imgExts {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	return false
}

type ytdlpMeta struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	URL         string            `json:"url"`
	Ext         string            `json:"ext"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	Duration    float64           `json:"duration"`
	FileSize    int64             `json:"filesize,omitempty"`
	FileSizeApp int64             `json:"filesize_approx,omitempty"`
	TBR         float64           `json:"tbr,omitempty"` 
	HttpHeaders map[string]string `json:"http_headers"`
	Formats     []ytdlpFormat     `json:"formats"`
}

type ytdlpFormat struct {
	ID     string `json:"format_id"`
	URL    string `json:"url"`
	Ext    string `json:"ext"`
	ACodec string `json:"acodec"`
	VCodec string `json:"vcodec"`
}
