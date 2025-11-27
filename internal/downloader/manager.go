package downloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/providers"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
)

func DownloadAudioWithProgress(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int, userID int64) ([]string, int64, string, error) {
	return downloadMedia(url, true, bot, chatID, msgID, "", userID)
}

// DownloadVideoWithProgress downloads video from the given URL.
func DownloadVideoWithProgress(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int, userID int64) ([]string, int64, string, error) {
	return downloadMedia(url, false, bot, chatID, msgID, "", userID)
}

// DownloadVideoWithProgressDetailed downloads video with detailed progress updates (e.g. for YouTube).
func DownloadVideoWithProgressDetailed(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string, userID int64) ([]string, int64, string, error) {
	return downloadMedia(url, false, bot, chatID, msgID, username, userID)
}

// downloadMedia is the unified function for downloading media.
func downloadMedia(url string, audioOnly bool, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string, userID int64) ([]string, int64, string, error) {
	mediaType := "Video"
	if audioOnly {
		mediaType = "Audio"
	}
	log.Printf("DownloadMedia: %s (type=%s, user=%d)", url, mediaType, userID)

	// Special handling for YouTube if we have username (implies detailed progress)
	if !audioOnly && isYouTubeURL(url) && username != "" {
		provider := providers.NewYouTubeProviderWithProgress(true, bot, chatID, msgID, username)
		filePaths, err := provider.Download(context.Background(), url, false)
		if err == nil && len(filePaths) > 0 {
			size := getTotalSize(filePaths)
			stats.GetStats().RecordDownload(userID, provider.Name(), mediaType, len(filePaths), size, true)
			return filePaths, size, provider.Name(), nil
		}
		// Fallback to standard providers if specific YouTube provider fails
		log.Printf("YouTube specific provider failed, falling back to standard providers: %v", err)
	}

	// Standard provider list
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

		filePaths, err := provider.Download(ctx, url, audioOnly)
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

		return filePaths, size, provider.Name(), nil
	}

	stats.GetStats().RecordDownload(userID, "Unknown", mediaType, 0, 0, false)
	if lastErr != nil {
		return nil, 0, "", lastErr
	}
	return nil, 0, "", fmt.Errorf("no suitable provider found or all failed")
}

// UniversalDownload is kept for backward compatibility or direct usage, simply calling downloadMedia.
func UniversalDownload(url string, audioOnly bool, userID int64) ([]string, int64, string, error) {
	return downloadMedia(url, audioOnly, nil, 0, 0, "", userID)
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
