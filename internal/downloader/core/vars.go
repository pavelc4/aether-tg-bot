package core

import "regexp"

var UnitMultipliers = map[string]float64{
	"B":     B,
	"KB":    KB,
	"KiB":   KB,
	"MB":    MB,
	"MiB":   MB,
	"GB":    GB,
	"GiB":   GB,
	"TB":    TB,
	"TiB":   TB,
	"B/s":   B,
	"KB/s":  KB,
	"KiB/s": KB,
	"MB/s":  MB,
	"MiB/s": MB,
	"GB/s":  GB,
	"GiB/s": GB,
	"TB/s":  TB,
	"TiB/s": TB,
}

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

var SizeUnits = []SizeUnit{
	{"TB", TB},
	{"GB", GB},
	{"MB", MB},
	{"KB", KB},
	{"B", B},
}

var YTDLPProgressRegex = regexp.MustCompile(`\[download\]\s+([\d.]+)%\s+of\s+~?\s*([^\s]+)(?:\s+in\s+([^\s]+)\s+at\s+([^\s]+)|(?:\s+at\s+([^\s]+)\s+ETA\s+(\S+)))`)
