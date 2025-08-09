package bot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func DownloadVideo(url string) (string, int64, string, error) {
	return runYTDLP(url, nil)
}

func DownloadAudio(url string) (string, int64, string, error) {
	extraArgs := []string{"-x", "--audio-format", "mp3"}
	return runYTDLP(url, extraArgs)
}

func runYTDLP(url string, extraArgs []string) (string, int64, string, error) {
	downloadPath, err := os.MkdirTemp("", "aether-dl-")
	if err != nil {
		return "", 0, "", fmt.Errorf("gagal membuat direktori sementara: %w", err)
	}

	outputTemplate := filepath.Join(downloadPath, "%(title)s.%(ext)s")
	if len(extraArgs) > 0 {
		outputTemplate = filepath.Join(downloadPath, "%(title)s.mp3")
	}

	args := []string{
		url,
		"-o", outputTemplate,
	}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	cmd := exec.Command("yt-dlp", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	fmt.Printf("Menjalankan perintah: yt-dlp %s\n", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(downloadPath)
		return "", 0, "", fmt.Errorf("gagal menjalankan yt-dlp: %w\nstderr: %s", err, stderr.String())
	}

	var newestFile string
	var newestTime time.Time

	files, err := os.ReadDir(downloadPath)
	if err != nil {
		return "", 0, "", fmt.Errorf("gagal membaca direktori unduhan '%s': %w", downloadPath, err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestFile = filepath.Join(downloadPath, file.Name())
		}
	}

	if newestFile == "" {
		_ = os.RemoveAll(downloadPath)
		return "", 0, "", fmt.Errorf("gagal menentukan file terbaru di direktori unduhan")
	}

	finalInfo, err := os.Stat(newestFile)
	if err != nil {
		return "", 0, "", fmt.Errorf("gagal stat file terbaru '%s': %w", newestFile, err)
	}

	provider := "yt-dlp"

	return newestFile, finalInfo.Size(), provider, nil
}
