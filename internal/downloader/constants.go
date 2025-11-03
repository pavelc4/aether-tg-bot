package downloader

import (
	"regexp"
	"time"
)

var (
	contentTypeToExt = map[string]string{
		"image/png":        ".png",
		"image/gif":        ".gif",
		"image/jpeg":       ".jpg",
		"video/mp4":        ".mp4",
		"video/webm":       ".webm",
		"video/quicktime":  ".mov",
		"video/x-matroska": ".mkv",
		"audio/mpeg":       ".mp3",
	}

	imageContentTypes = map[string]string{
		"image/png":  ".png",
		"image/gif":  ".gif",
		"image/jpeg": ".jpg",
	}

	ytdlpProgressRegex = regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*(\S+)\s+at\s+(\S+)\s+ETA\s+(\S+)`)
)

const (
	minFileSize     = 5120             // 5KB minimum
	downloadTimeout = 2 * time.Minute  // HTTP download timeout
	cobaltTimeout   = 60 * time.Second // Cobalt API timeout
	ytdlpTimeout    = 10 * time.Minute // yt-dlp execution timeout
)
