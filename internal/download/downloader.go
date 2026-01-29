package download

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type Downloader struct {
	streamMgr *streaming.Manager
	uploader  *telegram.Uploader
}

func NewDownloader(sm *streaming.Manager, upl *telegram.Uploader) *Downloader {
	return &Downloader{
		streamMgr: sm,
		uploader:  upl,
	}
}

func (d *Downloader) Download(ctx context.Context, infos []provider.VideoInfo, audioOnly bool) ([]tg.InputMediaClass, []provider.VideoInfo) {
	album := make([]tg.InputMediaClass, len(infos))
	uploadedInfos := make([]provider.VideoInfo, len(infos))

	var wg sync.WaitGroup

	for i, info := range infos {
		wg.Add(1)

		go func(i int, info provider.VideoInfo) {
			defer wg.Done()

			input := streaming.StreamInput{
				URL:      info.URL,
				Filename: info.FileName,
				Size:     info.FileSize,
				Headers:  info.Headers,
				MIME:     info.MimeType,
				Duration: info.Duration,
				Width:    info.Width,
				Height:   info.Height,
			}
			
			isHLS := strings.Contains(info.URL, ".m3u8") || strings.Contains(info.URL, ".mpd") || strings.Contains(info.URL, "manifest")
			
			if isHLS || info.UsePipe {
				logger.Info("Using piped download strategy", "url", info.URL, "file", info.FileName, "hls", isHLS, "pipe_flag", info.UsePipe)
				
				args := []string{
					"-o", "-",
					"--rm-cache-dir",
					"--js-runtimes", "bun",
					"--retries", "10",
					"--fragment-retries", "10",
					info.URL,
				}
				
				if info.UsePipe {
					args = append([]string{"-f", "bestvideo[height<=1080]+bestaudio/best[height<=1080]/bestvideo+bestaudio/best", "--merge-output-format", "mkv"}, args...)
					
					if !strings.HasSuffix(strings.ToLower(info.FileName), ".mkv") {
						ext := filepath.Ext(info.FileName)
						if ext != "" {
							info.FileName = strings.TrimSuffix(info.FileName, ext) + ".mkv"
						} else {
							info.FileName = info.FileName + ".mkv"
						}
						info.MimeType = "video/x-matroska"
					}
				}

				if cookies := config.GetYtdlpCookies(); cookies != "" {
					args = append([]string{"--cookies", cookies}, args...)
				}
				
				cmd := exec.CommandContext(ctx, "yt-dlp", args...)
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					logger.Error("Failed to create stdout pipe", "error", err)
					return
				}
				
				if err := cmd.Start(); err != nil {
					logger.Error("Failed to start yt-dlp pipe", "error", err, "stderr", stderr.String())
					return
				}
				
				// Log yt-dlp command for debugging
				logger.Info("Started yt-dlp pipe", "file", info.FileName, "args", args[:3])
				
				input.Reader = &cmdReader{
					ReadCloser: stdout,
					cmd:        cmd,
					stderr:     &stderr,
				}
			}

			isPhoto := strings.HasPrefix(input.MIME, "image/") ||
				strings.HasSuffix(strings.ToLower(input.Filename), ".jpg") ||
				strings.HasSuffix(strings.ToLower(input.Filename), ".jpeg") ||
				strings.HasSuffix(strings.ToLower(input.Filename), ".png") ||
				strings.HasSuffix(strings.ToLower(input.Filename), ".webp")

			// Use random ID for fileID to avoid collisions
			fileID := rand.Int63()
			isBig := !isPhoto && input.Size > 10*1024*1024

			logger.Info("Upload strategy",
				"file", input.Filename,
				"size", input.Size,
				"mime", input.MIME,
				"isPhoto", isPhoto,
				"isBig", isBig,
				"fileID", fileID,
			)

			uploadFn := func(ctx context.Context, chunk streaming.Chunk, _ int64) error {
				return d.uploader.UploadChunk(ctx, chunk, fileID, isBig)
			}

			actualParts, md5sum, err := d.streamMgr.Stream(ctx, input, uploadFn, func(read, total int64) {})

			if err != nil {
				logger.Error("Failed to stream item", "index", i, "error", err)
				return
			}

			if actualParts > 0 {
				media := CreateInputMedia(input, fileID, actualParts, isBig, md5sum, audioOnly)
				if media != nil {
					album[i] = media
					uploadedInfos[i] = info
				} else {
					logger.Error("Failed to create input media", "file", info.FileName)
				}
			}
		}(i, info)
	}

	wg.Wait()

	var finalAlbum []tg.InputMediaClass
	var finalInfos []provider.VideoInfo

	for i := range album {
		if album[i] != nil {
			finalAlbum = append(finalAlbum, album[i])
			finalInfos = append(finalInfos, uploadedInfos[i])
		}
	}

	return finalAlbum, finalInfos
}

type cmdReader struct {
	io.ReadCloser
	cmd    *exec.Cmd
	stderr *bytes.Buffer
}

func (c *cmdReader) Close() error {
	err := c.ReadCloser.Close()
	waitErr := c.cmd.Wait()

	stderrStr := c.stderr.String()
	if waitErr != nil {
		stderrLen := len(stderrStr)
		if stderrLen > 1000 {
			stderrStr = "..." + stderrStr[stderrLen-1000:]
		}
		
		logger.Error("Pipe process failed",
			"error", waitErr,
			"stderr", stderrStr,
		)
		return waitErr
	}
	// Log successful completion with stderr summary
	if len(stderrStr) > 0 {
		logger.Debug("Pipe completed", "stderr_lines", len(stderrStr)/100)
	}
	
	return err
}
