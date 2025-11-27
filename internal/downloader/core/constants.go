package core

import (
	"time"
)

const (
	MinFileSize     = 5120             // 5KB minimum
	DownloadTimeout = 2 * time.Minute  // HTTP download timeout
	CobaltTimeout   = 60 * time.Second // Cobalt API timeout
	YTDLPTimeout    = 10 * time.Minute // yt-dlp execution timeout
)

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)
