package messaging

import (
	"fmt"

	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/utils"
)

func FormatInitialProgress(infos []provider.VideoInfo, providerName string) string {
	engineDisplay := getEngineDisplay(providerName)

	if len(infos) == 0 {
		return fmt.Sprintf("🎥 Downloading... (Engine: %s)", engineDisplay)
	}

	totalSize := formatTotalSize(infos)
	title := infos[0].Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	return fmt.Sprintf("🎥 %s | %s | Engine: %s", title, totalSize, engineDisplay)
}

func formatTotalSize(infos []provider.VideoInfo) string {
	total := uint64(0)
	for _, info := range infos {
		total += uint64(info.FileSize)
	}
	return utils.FormatBytes(total)
}

func getEngineDisplay(providerName string) string {
	switch providerName {
	case "TikTok":
		return "TikWM API"
	case "YouTube":
		return "yt-dlp"
	case "Cobalt":
		return "Cobalt API"
	default:
		if providerName != "" {
			return providerName
		}
		return "yt-dlp + Bun"
	}
}
