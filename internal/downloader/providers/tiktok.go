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
	"strings"
	"time"

	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
)

func NewTikTokProvider() *TikTokProvider {
	return &TikTokProvider{
		timeout: tikTokTimeout,
		client:  pkghttp.GetDownloadClient(),
	}
}

func (tp *TikTokProvider) Name() string {
	return "TikTok"
}

func (tp *TikTokProvider) CanHandle(url string) bool {
	return strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com")
}

// Support BOTH audio and video
func (tp *TikTokProvider) Download(ctx context.Context, url string, audioOnly bool) ([]string, string, error) {
	if audioOnly {
		log.Printf(" TikTok: Downloading AUDIO")
		return tp.downloadAudio(ctx, url)
	} else {
		log.Printf(" TikTok: Downloading VIDEO")
		return tp.downloadVideo(ctx, url)
	}
}

func (tp *TikTokProvider) downloadVideo(ctx context.Context, url string) ([]string, string, error) {
	log.Printf(" TikTok: Fetching video URL...")
	response, err := tp.fetchVideoData(ctx, url)
	if err != nil {
		return nil, "", fmt.Errorf("fetch video data failed: %w", err)
	}

	if len(response.Data.Images) > 0 {
		log.Printf(" TikTok: Detected SLIDESHOW (%d images)", len(response.Data.Images))

		tmpDir, err := os.MkdirTemp("", "aether-tiktok-slides-")
		if err != nil {
			return nil, "", fmt.Errorf("create temp directory failed: %w", err)
		}

		imagePaths, err := tp.downloadImages(ctx, response.Data.Images, tmpDir)
		if err != nil {
			os.RemoveAll(tmpDir)
			return nil, "", fmt.Errorf("download images failed: %w", err)
		}

		return imagePaths, response.Data.Title, nil
	}

	// 2. Fallback to Video
	videoURL := ""
	if response.Data.Play != "" {
		videoURL = response.Data.Play
	}
	// TikWM returns relative URLs sometimes
	if videoURL != "" && !strings.HasPrefix(videoURL, "http") {
		videoURL = "https://tikwm.com" + videoURL
	}

	if videoURL == "" {
		return nil, "", fmt.Errorf("video URL not found")
	}

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-video-")
	if err != nil {
		return nil, "", fmt.Errorf("create temp directory failed: %w", err)
	}

	filePath, err := tp.downloadVideoFile(ctx, videoURL, tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("download video file failed: %w", err)
	}

	log.Printf(" TikTok: Video downloaded: %s", filePath)
	return []string{filePath}, response.Data.Title, nil
}

func (tp *TikTokProvider) downloadImages(ctx context.Context, urls []string, tmpDir string) ([]string, error) {
	var paths []string

	// Limit concurrency to avoid flooding
	// Simple sequential for now given TikTok slides usually < 35 images
	for i, imgURL := range urls {
		// Cleanup URL if needed
		if !strings.HasPrefix(imgURL, "http") {
			imgURL = "https://tikwm.com" + imgURL
		}

		req, err := http.NewRequestWithContext(ctx, "GET", imgURL, nil)
		if err != nil {
			log.Printf("Failed to create request for image %d: %v", i, err)
			continue
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := tp.client.Do(req)
		if err != nil {
			log.Printf("Failed to download image %d: %v", i, err)
			continue
		}
		defer func() {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Failed to download image %d (status %d)", i, resp.StatusCode)
			continue
		}

		// Determine extension from content-type or URL
		ext := ".jpg"
		if strings.Contains(imgURL, ".webp") {
			ext = ".webp"
		} else if strings.Contains(imgURL, ".png") {
			ext = ".png"
		}

		filePath := filepath.Join(tmpDir, fmt.Sprintf("slide_%d%s", i+1, ext))
		outFile, err := os.Create(filePath)
		if err != nil {
			log.Printf("Failed to create file for image %d: %v", i, err)
			continue
		}

		_, err = io.Copy(outFile, resp.Body)
		outFile.Close()
		if err != nil {
			log.Printf("Failed to save image %d: %v", i, err)
			os.Remove(filePath)
			continue
		}

		paths = append(paths, filePath)
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no images downloaded successfully")
	}

	return paths, nil
}

func (tp *TikTokProvider) downloadAudio(ctx context.Context, url string) ([]string, string, error) {
	log.Printf(" TikTok: Fetching audio URL...")
	audioURL, title, author, err := tp.fetchAudioURL(ctx, url)
	if err != nil {
		return nil, "", fmt.Errorf("fetch audio URL failed: %w", err)
	}

	log.Printf(" TikTok: Title=%s, Author=%s", title, author)

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-audio-")
	if err != nil {
		return nil, "", fmt.Errorf("create temp directory failed: %w", err)
	}

	filePath, err := tp.downloadAudioFile(ctx, audioURL, tmpDir, title)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, "", fmt.Errorf("download audio file failed: %w", err)
	}

	log.Printf(" TikTok: Audio downloaded successfully: %s", filePath)
	return []string{filePath}, title, nil
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
	audioURL := result.Data.Music
	if audioURL == "" {
		audioURL = music.Play
	}

	if audioURL != "" && !strings.HasPrefix(audioURL, "http") {
		audioURL = "https://tikwm.com" + audioURL
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
