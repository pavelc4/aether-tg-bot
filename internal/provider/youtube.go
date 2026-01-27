package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	formatArg := "best[ext=mp4][protocol^=http]/best[protocol^=http]"
	if opts.AudioOnly {
		formatArg = "bestaudio[ext=m4a]/bestaudio"
	}

	args := []string{
		"--dump-json",
		"--no-playlist",
		"--no-warnings",
		"-f", formatArg,
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
	if (finalURL == "" || isNonStreamableURL(finalURL)) && len(meta.Formats) > 0 {
		// Find a valid video URL in formats
		for _, f := range meta.Formats {
			if f.URL != "" && f.VCodec != "none" && !isNonStreamableURL(f.URL) {
				finalURL = f.URL
				break
			}
		}
	}

	if finalURL == "" || isNonStreamableURL(finalURL) {
		logger.Error("yt-dlp returned no streamable video URL", "url", finalURL, "stderr", stderr.String())
		return nil, fmt.Errorf("no streamable video URL found")
	}

	filename := fmt.Sprintf("%s.%s", meta.Title, meta.Ext)
	filename = strings.ReplaceAll(filename, "/", "_")

	size := meta.FileSize
	if size == 0 {
		size = meta.FileSizeApp
	}

	if size == 0 && finalURL != "" {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		
		req, _ := http.NewRequestWithContext(ctx, "HEAD", finalURL, nil)
		for k, v := range meta.HttpHeaders {
			req.Header.Set(k, v)
		}
		
		if resp, err := http.DefaultClient.Do(req); err == nil {
			size = resp.ContentLength
			resp.Body.Close()
		}
	}

	logger.Info("YouTube info resolved", 
		"title", meta.Title, 
		"size", size,
		"res", fmt.Sprintf("%dx%d", meta.Width, meta.Height),
		"dur", meta.Duration,
	)
	mime := "video/" + meta.Ext
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
	}}, nil
}

func isNonStreamableURL(url string) bool {
	lower := strings.ToLower(url)
	if strings.Contains(lower, ".jpg") || strings.Contains(lower, ".jpeg") || strings.Contains(lower, ".png") || strings.Contains(lower, ".webp") {
		return true
	}
	if strings.Contains(lower, ".m3u8") || strings.Contains(lower, ".mpd") || strings.Contains(lower, "manifest") {
		return true
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
	HttpHeaders map[string]string `json:"http_headers"`
	Formats     []ytdlpFormat     `json:"formats"`
}

type ytdlpFormat struct {
	URL    string `json:"url"`
	VCodec string `json:"vcodec"`
}
