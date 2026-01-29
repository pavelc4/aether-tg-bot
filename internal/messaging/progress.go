package messaging

import (
	"fmt"

	"github.com/pavelc4/aether-tg-bot/internal/provider"
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
	return formatBytes(total)
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	val := int64(b)
	for n := val / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(val)/float64(div), "KMGTPE"[exp])
}
