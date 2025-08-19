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
			PlayURL string `json:"play"`
			Author  string `json:"author"`
		} `json:"music_info"`
	} `json:"data"`
}

func DownloadTikTokAudio(tiktokURL string) (filePath, title, author string, err error) {
	log.Println("Memulai proses unduh audio TikTok dari:", tiktokURL)

	apiURL := "https://www.tikwm.com/api/"
	payload := map[string]string{"url": tiktokURL}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", "", "", fmt.Errorf("gagal memanggil API tikwm: %w", err)
	}
	defer resp.Body.Close()

	var apiResp TikWMResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", "", "", fmt.Errorf("gagal decode JSON dari tikwm: %w", err)
	}

	musicInfo := apiResp.Data.MusicInfo
	if musicInfo.PlayURL == "" {
		return "", "", "", fmt.Errorf("tidak ditemukan URL audio di response API")
	}

	tmpDir, err := os.MkdirTemp("", "aether-tiktok-audio-")
	if err != nil {
		return "", "", "", fmt.Errorf("gagal membuat direktori sementara: %w", err)
	}

	audioResp, err := http.Get(musicInfo.PlayURL)
	if err != nil {
		DeleteDirectory(tmpDir)
		return "", "", "", fmt.Errorf("gagal mengunduh audio: %w", err)
	}
	defer audioResp.Body.Close()

	fileName := musicInfo.ID + ".mp3"
	if musicInfo.Title != "" {
		safeTitle := strings.ReplaceAll(musicInfo.Title, " ", "_")
		safeTitle = strings.ToLower(safeTitle)
		fileName = safeTitle + ".mp3"
	}

	filePath = filepath.Join(tmpDir, fileName)
	outFile, err := os.Create(filePath)
	if err != nil {
		DeleteDirectory(tmpDir)
		return "", "", "", fmt.Errorf("gagal membuat file audio: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, audioResp.Body)
	if err != nil {
		DeleteDirectory(tmpDir)
		return "", "", "", fmt.Errorf("gagal menyimpan file audio: %w", err)
	}

	log.Printf("Audio TikTok berhasil diunduh ke: %s", filePath)
	return filePath, musicInfo.Title, musicInfo.Author, nil
}
