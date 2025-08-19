package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TikWMResponse struct {
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

func fetchAudioURL(tiktokURL string) (string, string, string, error) {
	apiURL := "https://www.tikwm.com/api/"
	payload := map[string]string{"url": tiktokURL}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", "", "", fmt.Errorf("gagal call API: %w", err)
	}
	defer resp.Body.Close()

	var result TikWMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", "", fmt.Errorf("decode JSON gagal: %w", err)
	}

	music := result.Data.MusicInfo
	audioURL := music.Play
	if audioURL == "" {
		audioURL = music.PlayURL
	}
	if audioURL == "" {
		return "", "", "", fmt.Errorf("URL audio tidak ditemukan di response")
	}

	return audioURL, music.Title, music.Author, nil
}

func DownloadTikTokAudio(tiktokURL string) (filePath, title, author string, err error) {
	audioURL, title, author, err := fetchAudioURL(tiktokURL)
	if err != nil {
		return "", "", "", err
	}

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-audio-")
	if err != nil {
		return "", "", "", fmt.Errorf("gagal membuat direktori sementara: %w", err)
	}

	resp, err := http.Get(audioURL)
	if err != nil {
		DeleteDirectory(tmpDir)
		return "", "", "", fmt.Errorf("gagal mengunduh audio: %w", err)
	}
	defer resp.Body.Close()

	safeTitle := "tiktok_audio"
	if title != "" {
		safeTitle = strings.ReplaceAll(strings.ToLower(title), " ", "_")
	}
	filePath = filepath.Join(tmpDir, safeTitle+".mp3")

	outFile, err := os.Create(filePath)
	if err != nil {
		DeleteDirectory(tmpDir)
		return "", "", "", fmt.Errorf("gagal membuat file audio: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		DeleteDirectory(tmpDir)
		return "", "", "", fmt.Errorf("gagal menyimpan file audio: %w", err)
	}

	log.Printf("Audio TikTok berhasil diunduh: %s", filePath)
	return filePath, title, author, nil
}
