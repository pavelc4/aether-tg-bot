package downloader

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
)

const (
	tikTokAPIURL   = "https://www.tikwm.com/api/"
	tikTokTimeout  = 30 * time.Second
	maxFilenameLen = 200
	minAudioSize   = 5120 // 5KB minimum
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
	} `json:"data"`
}

func DownloadTikTokAudio(tiktokURL string) (filePath, title, author string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), tikTokTimeout)
	defer cancel()

	audioURL, title, author, err := fetchAudioURL(ctx, tiktokURL)
	if err != nil {
		return "", "", "", err
	}

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-audio-")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	defer func() {
		if err != nil {
			os.RemoveAll(tmpDir)
		}
	}()

	filePath, err = downloadAudioFile(ctx, audioURL, tmpDir, title)
	if err != nil {
		return "", "", "", err
	}

	log.Printf("TikTok audio downloaded successfully: %s", filePath)
	return filePath, title, author, nil
}

func fetchAudioURL(ctx context.Context, tiktokURL string) (string, string, string, error) {
	payload := map[string]string{"url": tiktokURL}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tikTokAPIURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := GetDownloadClient().Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to call API: %w", err)
	}

	defer func() {
		io.Copy(io.Discard, resp.Body) // Drain body for connection reuse
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result TikWMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", "", fmt.Errorf("JSON decoding failed: %w", err)
	}

	if result.Code != 0 {
		return "", "", "", fmt.Errorf("API error: %s", result.Msg)
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

func downloadAudioFile(ctx context.Context, audioURL, tmpDir, title string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := GetDownloadClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download audio: %w", err)
	}

	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	safeFilename := sanitizeFilename(title)
	if safeFilename == "" {
		safeFilename = "tiktok_audio"
	}

	filePath := filepath.Join(tmpDir, safeFilename+".mp3")
	outFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create audio file: %w", err)
	}

	defer outFile.Close()

	size, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save audio file: %w", err)
	}

	if size < minAudioSize {
		return "", fmt.Errorf("audio file is too small (%d bytes), possibly invalid", size)
	}

	return filePath, nil
}

func sanitizeFilename(filename string) string {
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
