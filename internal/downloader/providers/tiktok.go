package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	httpclient "github.com/pavelc4/aether-tg-bot/pkg/http"
)

const (
	tikTokAPIURL   = "https://www.tikwm.com/api/"
	tikTokTimeout  = 30 * time.Second
	maxFilenameLen = 200
	minAudioSize   = 5120
	minVideoSize   = 102400 // 100KB minimum for video
)

var (
	unsafeCharsRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	spacesRegex      = regexp.MustCompile(`\s+`)
)

type TikWMResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MusicInfo struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Author  string `json:"author"`
			Play    string `json:"play"`
			PlayURL string `json:"play_url"`
		} `json:"music_info"`
		VideoInfo struct {
			DownloadAddr string `json:"downloadAddr"`
			PlayAddr     string `json:"playAddr"`
		} `json:"video"`
		Videos []struct {
			DownloadAddr string `json:"downloadAddr"`
			PlayAddr     string `json:"playAddr"`
		} `json:"videos"`
	} `json:"data"`
}

type TikTokProvider struct {
	timeout time.Duration
	client  *http.Client
}

func NewTikTokProvider() *TikTokProvider {
	return &TikTokProvider{
		timeout: tikTokTimeout,
		client:  httpclient.GetDownloadClient(),
	}
}

func (tp *TikTokProvider) Name() string {
	return "TikTok"
}

func (tp *TikTokProvider) CanHandle(url string) bool {
	return strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com")
}

// Support BOTH audio and video
func (tp *TikTokProvider) Download(ctx context.Context, url string, audioOnly bool) ([]string, error) {
	if audioOnly {
		log.Printf(" TikTok: Downloading AUDIO")
		return tp.downloadAudio(ctx, url)
	} else {
		log.Printf(" TikTok: Downloading VIDEO")
		return tp.downloadVideo(ctx, url)
	}
}

func (tp *TikTokProvider) downloadVideo(ctx context.Context, url string) ([]string, error) {
	log.Printf(" TikTok: Fetching video URL...")
	response, err := tp.fetchVideoData(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch video data failed: %w", err)
	}

	// Get video URL from response
	videoURL := ""
	if response.Data.VideoInfo.DownloadAddr != "" {
		videoURL = response.Data.VideoInfo.DownloadAddr
	}
	if videoURL == "" && len(response.Data.Videos) > 0 {
		videoURL = response.Data.Videos[0].DownloadAddr
	}

	if videoURL == "" {
		return nil, fmt.Errorf("video URL not found")
	}

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-video-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory failed: %w", err)
	}

	filePath, err := tp.downloadVideoFile(ctx, videoURL, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("download video file failed: %w", err)
	}

	log.Printf(" TikTok: Video downloaded: %s", filePath)
	return []string{filePath}, nil
}

func (tp *TikTokProvider) downloadAudio(ctx context.Context, url string) ([]string, error) {
	log.Printf(" TikTok: Fetching audio URL...")
	audioURL, title, author, err := tp.fetchAudioURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch audio URL failed: %w", err)
	}

	log.Printf(" TikTok: Title=%s, Author=%s", title, author)

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-audio-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory failed: %w", err)
	}

	filePath, err := tp.downloadAudioFile(ctx, audioURL, tmpDir, title)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("download audio file failed: %w", err)
	}

	log.Printf(" TikTok: Audio downloaded successfully: %s", filePath)
	return []string{filePath}, nil
}

func (tp *TikTokProvider) fetchVideoData(ctx context.Context, tiktokURL string) (*TikWMResponse, error) {
	payload := map[string]string{"url": tiktokURL}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tikTokAPIURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := tp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result TikWMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Msg)
	}

	return &result, nil
}

func (tp *TikTokProvider) downloadVideoFile(ctx context.Context, videoURL, tmpDir string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := tp.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	filePath := filepath.Join(tmpDir, fmt.Sprintf("tiktok_%d.mp4", time.Now().UnixNano()))
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer outFile.Close()

	size, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("save file failed: %w", err)
	}

	if size < minVideoSize {
		return "", fmt.Errorf("video file too small (%d bytes)", size)
	}

	return filePath, nil
}

func (tp *TikTokProvider) fetchAudioURL(ctx context.Context, tiktokURL string) (string, string, string, error) {
	result, err := tp.fetchVideoData(ctx, tiktokURL)
	if err != nil {
		return "", "", "", err
	}

	music := result.Data.MusicInfo
	audioURL := music.Play
	if audioURL == "" {
		audioURL = music.PlayURL
	}

	if audioURL == "" {
		return "", "", "", fmt.Errorf("audio URL not found in response")
	}

	return audioURL, music.Title, music.Author, nil
}

func (tp *TikTokProvider) downloadAudioFile(ctx context.Context, audioURL, tmpDir, title string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	resp, err := tp.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	safeFilename := tp.sanitizeFilename(title)
	if safeFilename == "" {
		safeFilename = "tiktok_audio"
	}

	filePath := filepath.Join(tmpDir, safeFilename+".mp3")
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer outFile.Close()

	size, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("save file failed: %w", err)
	}

	if size < minAudioSize {
		return "", fmt.Errorf("audio file too small (%d bytes)", size)
	}

	return filePath, nil
}

func (tp *TikTokProvider) sanitizeFilename(filename string) string {
	if filename == "" {
		return ""
	}

	filename = filepath.Base(filename)
	filename = strings.ToLower(filename)
	filename = spacesRegex.ReplaceAllString(filename, "_")
	filename = unsafeCharsRegex.ReplaceAllString(filename, "")
	filename = strings.Trim(filename, ". ")
	filename = strings.ReplaceAll(filename, " ", "_")

	if len(filename) > maxFilenameLen {
		filename = filename[:maxFilenameLen]
	}

	if filename == "" || filename == "." || filename == ".." {
		return ""
	}

	return filename
}
