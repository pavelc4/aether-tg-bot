package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type YTDLPMetadata struct {
	Extractor string `json:"extractor"`
}

func DownloadVideo(url string) (string, int64, string, error) {
	return runYTDLP(url, "")
}

func DownloadAudio(url string) (string, int64, string, error) {
	return runYTDLP(url, "--extract-audio --audio-format mp3")
}

func runYTDLP(url, extraArgs string) (string, int64, string, error) {

	cmdMeta := exec.Command("yt-dlp", "-j", url)
	var metaOut bytes.Buffer
	cmdMeta.Stdout = &metaOut
	if err := cmdMeta.Run(); err != nil {
		return "", 0, "", fmt.Errorf("gagal ambil metadata: %w", err)
	}

	var meta YTDLPMetadata
	if err := json.Unmarshal(metaOut.Bytes(), &meta); err != nil {
		return "", 0, "", fmt.Errorf("gagal parse metadata: %w", err)
	}

	// Download file
	outputTemplate := "%(title)s.%(ext)s"
	args := []string{"-o", outputTemplate}
	if extraArgs != "" {
		args = append(args, strings.Split(extraArgs, " ")...)
	}
	args = append(args, url)

	cmd := exec.Command("yt-dlp", args...)
	if err := cmd.Run(); err != nil {
		return "", 0, "", fmt.Errorf("gagal download: %w", err)
	}

	matches, _ := filepath.Glob("*")
	var latestFile string
	var latestSize int64
	for _, f := range matches {
		if strings.HasSuffix(f, ".mp4") || strings.HasSuffix(f, ".mkv") || strings.HasSuffix(f, ".mp3") {
			info, _ := os.Stat(f)
			if info.Size() > latestSize {
				latestFile = f
				latestSize = info.Size()
			}
		}
	}

	return latestFile, latestSize, meta.Extractor, nil
}
