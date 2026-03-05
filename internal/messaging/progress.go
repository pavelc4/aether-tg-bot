package messaging

import (
	"fmt"

	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/utils"
)

func FormatInitialProgress(infos []provider.VideoInfo, providerName string) string {
	engineDisplay := getEngineDisplay(providerName)

	if len(infos) == 0 {
		return fmt.Sprintf("🎥 <b>Download</b>\n\n┌ Status : <code>Starting...</code>\n└ Engine : <code>%s</code>", engineDisplay)
	}

	totalSize := formatTotalSize(infos)
	title := infos[0].Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	return fmt.Sprintf(
		"🎥 <b>%s</b>\n\n"+
			"┌ Status : <code>Starting...</code>\n"+
			"├ [<code>□□□□□□□□□□□□</code>]\n"+
			"├ Ukuran : <code>%s</code>\n"+
			"├ Diproses : <code>0 B</code>\n"+
			"├ Kecepatan : <code>-</code>\n"+
			"├ Waktu : <code>0s</code>\n"+
			"└ Engine : <code>%s</code>",
		title,
		totalSize,
		engineDisplay,
	)
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
