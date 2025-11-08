package core

import (
	"regexp"
	"time"
)

const (
	MinFileSize     = 5120             // 5KB minimum
	DownloadTimeout = 2 * time.Minute  // HTTP download timeout
	CobaltTimeout   = 60 * time.Second // Cobalt API timeout
	YTDLPTimeout    = 10 * time.Minute // yt-dlp execution timeout
)

var ContentTypeToExt = map[string]string{
	"image/png":        ".png",
	"image/gif":        ".gif",
	"image/jpeg":       ".jpg",
	"video/mp4":        ".mp4",
	"video/webm":       ".webm",
	"video/quicktime":  ".mov",
	"video/x-matroska": ".mkv",
	"audio/mpeg":       ".mp3",
}

var ImageContentTypes = map[string]string{
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/jpeg": ".jpg",
}

var YTDLPProgressRegex = regexp.MustCompile(`\[download\]\s+([\d.]+)%\s+of\s+~?\s*([^\s]+)(?:\s+in\s+([^\s]+)\s+at\s+([^\s]+)|(?:\s+at\s+([^\s]+)\s+ETA\s+(\S+)))`)
