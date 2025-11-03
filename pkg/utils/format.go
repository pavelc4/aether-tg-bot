package utils

import (
	"fmt"
	"strings"
)

const (
	progressBarFilled = "█"
	progressBarEmpty  = "░"
	progressBarLength = 20
)

var markdownV2Replacer = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]",
	"(", "\\(", ")", "\\)", "~", "\\~", "`", "\\`",
	">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-",
	"=", "\\=", "|", "\\|", "{", "\\{", "}", "\\}",
	".", "\\.", "!", "\\!",
)

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func FormatDuration(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, secs)
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, secs)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

func EscapeMarkdownV2(s string) string {
	return markdownV2Replacer.Replace(s)
}

func FormatProgressBar(progress float64) string {
	if progress > 100 {
		progress = 100
	}
	if progress < 0 {
		progress = 0
	}

	filled := int(progress / 100 * progressBarLength)
	empty := progressBarLength - filled

	return fmt.Sprintf(
		"%s%s %.1f%%",
		strings.Repeat(progressBarFilled, filled),
		strings.Repeat(progressBarEmpty, empty),
		progress,
	)
}
