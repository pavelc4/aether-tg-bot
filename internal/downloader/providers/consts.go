package providers

import "time"

const (
	tikTokAPIURL    = "https://www.tikwm.com/api/"
	tikTokTimeout   = 30 * time.Second // TikTok timeout
	maxFilenameLen  = 200
	minAudioSize    = 5120
	minVideoSize    = 102400           // 100KB minimum for video
	cobaltTimeout   = 60 * time.Second // Cobalt timeout
	downloadTimeout = 2 * time.Minute  // Download timeout
	minFileSize     = 5120
	youtubeTimeout  = 10 * time.Minute // Youtube time out
)
