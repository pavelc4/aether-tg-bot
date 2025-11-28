package providers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pavelc4/aether-tg-bot/internal/downloader/core"
)

func NewYouTubeProvider(useCookies bool) *YouTubeProvider {
	return &YouTubeProvider{
		timeout:    youtubeTimeout,
		useCookies: useCookies,
	}
}

func (yp *YouTubeProvider) Name() string {
	return "YouTube (yt-dlp)"
}

func (yp *YouTubeProvider) CanHandle(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

func (yp *YouTubeProvider) Download(ctx context.Context, url string, audioOnly bool) ([]string, error) {
	log.Printf("YouTube: Starting download (audio=%v, cookies=%v)", audioOnly, yp.useCookies)

	tmpDir, err := os.MkdirTemp("", "aether-youtube-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory failed: %w", err)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), yp.timeout)
		defer cancel()
	}

	args := yp.buildArgs(tmpDir, audioOnly)
	args = append(args, url)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	go yp.trackProgress(stdoutPipe)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Printf("YouTube: Executing yt-dlp with %d args", len(args))

	if err := cmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("yt-dlp failed: %w\nStderr: %s", err, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(tmpDir, "*"))
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("list files failed: %w", err)
	}

	if len(files) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no files downloaded")
	}

	var validFiles []string
	for _, f := range files {
		if !strings.HasPrefix(filepath.Base(f), ".") {
			validFiles = append(validFiles, f)
		}
	}

	if len(validFiles) == 0 {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("no valid files found")
	}

	log.Printf("YouTube: Downloaded %d file(s)", len(validFiles))
	return validFiles, nil
}

func (yp *YouTubeProvider) trackProgress(stdoutPipe io.ReadCloser) {
	defer stdoutPipe.Close()

	scanner := bufio.NewScanner(stdoutPipe)
	buf := make([]byte, 256*1024)
	scanner.Buffer(buf, 256*1024)

	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if strings.Contains(line, "[download]") && strings.Contains(line, "%") {
			matches := core.YTDLPProgressRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				log.Printf("YouTube Progress: %s%%", matches[1])
			}
		}
	}
}

func (yp *YouTubeProvider) buildArgs(tmpDir string, audioOnly bool) []string {
	format := "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best"
	if audioOnly {
		format = "bestaudio/best"
	}

	args := []string{
		"-f", format,
		"-o", filepath.Join(tmpDir, "%(title)s.%(ext)s"),
		"--no-playlist",
		"--socket-timeout", "60",
		"--retries", "5",
		"--fragment-retries", "5",
		"--retry-sleep", "3",
		"--abort-on-error",
		"--no-warnings",
		"-N", "16",
		"--newline",
		"--progress",
	}

	if audioOnly {
		args = append(args, "-x", "--audio-format", "mp3")
	}

	if yp.useCookies {
		cookiePath := yp.getCookiePath()
		if _, err := os.Stat(cookiePath); err == nil {
			args = append(args, "--cookies", cookiePath)
			log.Printf("YouTube: Using cookies from: %s", cookiePath)
		}
	}

	return args
}

func (yp *YouTubeProvider) getCookiePath() string {
	if path := os.Getenv("YTDLP_COOKIES"); path != "" {
		return path
	}
	return "cookies.txt"
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseSize(sizeStr string) (float64, string) {
	re := regexp.MustCompile(`([\d.]+)([A-Za-z]+)`)
	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) >= 3 {
		return parseFloat(matches[1]), matches[2]
	}
	return parseFloat(sizeStr), ""
}
