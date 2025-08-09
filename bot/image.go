package bot

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func DownloadImagesFromURL(url string) ([]string, error) {
	downloadPath, err := os.MkdirTemp("", "aether-img-")
	if err != nil {
		return nil, fmt.Errorf("gagal membuat direktori sementara: %w", err)
	}

	var args []string
	cookiePath := getCookieForURL(url)

	if cookiePath != "" {
		fmt.Printf("Menggunakan cookie: %s\n", cookiePath)
		args = []string{"--cookies", cookiePath, "--directory", downloadPath, url}
	} else {
		fmt.Println("Mencoba mengunduh tanpa cookie...")
		args = []string{"--directory", downloadPath, url}
	}

	cmd := exec.Command("gallery-dl", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	fmt.Printf("Menjalankan perintah: gallery-dl %s\n", strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(downloadPath)
		return nil, fmt.Errorf("gagal menjalankan gallery-dl: %w\nstderr: %s", err, stderr.String())
	}

	files, err := os.ReadDir(downloadPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca direktori hasil unduhan: %w", err)
	}

	var downloadedFiles []string
	for _, file := range files {
		if !file.IsDir() {
			downloadedFiles = append(downloadedFiles, filepath.Join(downloadPath, file.Name()))
		}
	}

	if len(downloadedFiles) == 0 {
		_ = os.RemoveAll(downloadPath)
		return nil, fmt.Errorf("tidak ada gambar yang berhasil diunduh")
	}

	return downloadedFiles, nil
}

func getCookieForURL(url string) string {
	var cookieFilename string

	switch {
	case strings.Contains(url, "instagram.com"):
		cookieFilename = "instagram-cookies.txt"
	case strings.Contains(url, "twitter.com") || strings.Contains(url, "x.com"):
		cookieFilename = "twitter-cookies.txt"
	case strings.Contains(url, "facebook.com"):
		cookieFilename = "facebook-cookies.txt"
	case strings.Contains(url, "tiktok.com"):
		cookieFilename = "tiktok-cookies.txt"
	default:
		return ""
	}

	if _, err := os.Stat(cookieFilename); err == nil {
		return cookieFilename
	}

	return ""
}
