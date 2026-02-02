package messaging

import (
	"fmt"

	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/utils"
)

func FormatInitialProgress(infos []provider.VideoInfo) string {
	if len(infos) == 0 {
		return "🎥 <b>Download</b>\n┌ Status : <code>Starting...</code>\n└ Engine : <code>yt-dlp + Bun</code>"
	}

	totalSize := formatTotalSize(infos)
	title := infos[0].Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	return fmt.Sprintf(
		"🎥 <b>%s</b>\n"+
			"┌ Status : <code>Starting...</code>\n"+
			"├ [<code>□□□□□□□□□□□□</code>]\n"+
			"├ Ukuran : <code>%s</code>\n"+
			"├ Diproses : <code>0 B</code>\n"+
			"├ Kecepatan : <code>-</code>\n"+
			"├ Waktu : <code>0s</code>\n"+
			"└ Engine : <code>yt-dlp + Bun</code>",
		title,
		totalSize,
	)
}

func formatTotalSize(infos []provider.VideoInfo) string {
	total := uint64(0)
	for _, info := range infos {
		total += uint64(info.FileSize)
	}
	return utils.FormatBytes(total)
}
