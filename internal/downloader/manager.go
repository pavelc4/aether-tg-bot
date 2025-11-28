package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/pavelc4/aether-tg-bot/internal/downloader/providers"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
)

func UniversalDownload(url string, audioOnly bool, userID int64) ([]string, int64, string, string, error) {
	mediaType := "Video"
	if audioOnly {
		mediaType = "Audio"
	}
	log.Printf("DownloadMedia: %s (type=%s, user=%d)", url, mediaType, userID)

	providersList := []providers.Provider{
		providers.NewCobaltProvider(),
		providers.NewTikTokProvider(),
		providers.NewYouTubeProvider(true),
	}

	ctx := context.Background()
	var lastErr error

	for _, provider := range providersList {
		if !provider.CanHandle(url) {
			continue
		}

		log.Printf("Trying %s provider (audioOnly=%v)", provider.Name(), audioOnly)

		filePaths, title, err := provider.Download(ctx, url, audioOnly)
		if err != nil {
			log.Printf("%s failed: %v", provider.Name(), err)
			lastErr = err
			continue
		}

		if len(filePaths) == 0 {
			log.Printf("%s: No files downloaded", provider.Name())
			continue
		}

		size := getTotalSize(filePaths)
		log.Printf("%s: Successfully downloaded %d file(s)", provider.Name(), len(filePaths))
		stats.GetStats().RecordDownload(userID, provider.Name(), mediaType, len(filePaths), size, true)

		return filePaths, size, provider.Name(), title, nil
	}

	stats.GetStats().RecordDownload(userID, "Unknown", mediaType, 0, 0, false)
	if lastErr != nil {
		return nil, 0, "", "", lastErr
	}
	return nil, 0, "", "", fmt.Errorf("no suitable provider found or all failed")
}

func getTotalSize(filePaths []string) int64 {
	var total int64
	for _, path := range filePaths {
		if info, err := os.Stat(path); err == nil {
			total += info.Size()
		}
	}
	return total
}

func isYouTubeURL(url string) bool {
	return (url != "") && (strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be"))
}
