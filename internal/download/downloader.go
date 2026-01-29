package download

import (
	"context"
	"math/rand"
	"strings"
	"sync"

	"github.com/gotd/td/tg"
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
