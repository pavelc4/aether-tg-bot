package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/pavelc4/aether-tg-bot/internal/stats"
)

func formatPlatform(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	for _, platformName := range platformMap {
		if strings.ToLower(platformName) == source {
			return platformName
		}
	}
	if source != "" {
		return strings.ToUpper(source[:1]) + source[1:]
	}
	return "Unknown"
}

func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
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

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

func formatUptime(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func BuildMediaCaption(source, url, mediaType string, size int64, duration time.Duration, username string) string {
	caption := fmt.Sprintf(
		"*Media Downloaded Successfully*\n\n"+
			"🔗 *Source :* [%s](%s)\n"+
			"💾 *Size :* `%s`\n"+
			"⏱️ *Processing Time :* `%s`\n"+
			"👤 *By :* @%s",
		formatPlatform(source),
		url,
		FormatFileSize(size),
		formatDuration(duration),
		username,
	)
	return caption
}

func formatPeriodStats(s *stats.PeriodStats) string {
	if s == nil {
		return "No data"
	}
	return fmt.Sprintf("%d downloads (%s)", s.Downloads, FormatFileSize(s.Bytes))
}
