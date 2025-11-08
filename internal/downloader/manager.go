package downloader

import (
	"context"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/providers"
)

// DownloadAudioWithProgress downloads audio with progress tracking
func DownloadAudioWithProgress(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, int64, string, error) {
	log.Printf("DownloadAudioWithProgress: %s", url)

	filePaths, provider, err := downloadWithProviders(url, true)
	if err != nil {
		return nil, 0, "", fmt.Errorf("audio download failed: %w", err)
	}

	size := getTotalSize(filePaths)
	return filePaths, size, provider, nil
}

// DownloadVideoWithProgress downloads video with progress tracking
func DownloadVideoWithProgress(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int) ([]string, int64, string, error) {
	log.Printf("DownloadVideoWithProgress: %s", url)

	filePaths, provider, err := downloadWithProviders(url, false)
	if err != nil {
		return nil, 0, "", fmt.Errorf("video download failed: %w", err)
	}

	size := getTotalSize(filePaths)
	return filePaths, size, provider, nil
}

// DownloadVideoWithProgressDetailed downloads video dengan detailed progress tracking
func DownloadVideoWithProgressDetailed(url string, bot *tgbotapi.BotAPI, chatID int64, msgID int, username string) ([]string, int64, string, error) {
	log.Printf("DownloadVideoWithProgressDetailed: %s (user: %s)", url, username)

	// YouTube dengan progress tracking detail
	if isYouTubeURL(url) {
		provider := providers.NewYouTubeProviderWithProgress(true, bot, chatID, msgID, username)
		filePaths, err := provider.Download(context.Background(), url, false)
		if err == nil && len(filePaths) > 0 {
			size := getTotalSize(filePaths)
			return filePaths, size, provider.Name(), nil
		}
	}

	filePaths, provider, err := downloadWithProviders(url, false)
	if err != nil {
		return nil, 0, "", fmt.Errorf("video download failed: %w", err)
	}

	size := getTotalSize(filePaths)
	return filePaths, size, provider, nil
}

// downloadWithProviders tries different providers to download content
func downloadWithProviders(url string, audioOnly bool) ([]string, string, error) {
	ctx := context.Background()
	providersList := []providers.Provider{
		providers.NewCobaltProvider(),
		providers.NewTikTokProvider(),
		providers.NewYouTubeProvider(true),
	}

	var lastErr error
	for _, provider := range providersList {
		if !provider.CanHandle(url) {
			log.Printf("%s: Can't handle URL", provider.Name())
			continue
		}

		log.Printf("Trying %s provider", provider.Name())

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

		log.Printf("%s: Successfully downloaded %d file(s)", provider.Name(), len(filePaths))
		return filePaths, provider.Name(), nil
	}

	if lastErr != nil {
		return nil, "", lastErr
	}

	return nil, "", fmt.Errorf("no suitable provider found for URL: %s", url)
}

// getTotalSize calculates total size from all downloaded files
func getTotalSize(filePaths []string) int64 {
	var total int64
	for _, path := range filePaths {
		if info, err := os.Stat(path); err == nil {
			total += info.Size()
		}
	}
	return total
}

// UniversalDownload downloads content dari any URL
func UniversalDownload(url string, audioOnly bool) ([]string, int64, string, error) {
	ctx := context.Background()

	providersList := []providers.Provider{
		providers.NewCobaltProvider(),
		providers.NewTikTokProvider(),
		providers.NewYouTubeProvider(true),
	}

	for _, provider := range providersList {
		if !provider.CanHandle(url) {
			continue
		}

		log.Printf("Trying %s provider (audioOnly=%v)", provider.Name(), audioOnly)

		filePaths, err := provider.Download(ctx, url, audioOnly)
		if err == nil && len(filePaths) > 0 {
			size := getTotalSize(filePaths)
			log.Printf("%s: Downloaded %d file(s)", provider.Name(), len(filePaths))
			return filePaths, size, provider.Name(), nil
		}
	}

	return nil, 0, "", fmt.Errorf("download failed")
}

// isYouTubeURL checks if URL is YouTube
func isYouTubeURL(url string) bool {
	return (url != "") && (containsString(url, "youtube.com") || containsString(url, "youtu.be"))
}

// containsString helper function
func containsString(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
