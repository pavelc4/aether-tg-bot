package core

import (
	"log"
	"time"
)

type DownloadMetrics struct {
	StartTime  time.Time
	EndTime    time.Time
	FileSize   int64
	FilesCount int
	Provider   string
	Duration   time.Duration
	AvgSpeed   float64
}

func (m *DownloadMetrics) LogMetrics() {
	m.Duration = m.EndTime.Sub(m.StartTime)
	if m.Duration.Seconds() > 0 {
		m.AvgSpeed = float64(m.FileSize/(1024*1024)) / m.Duration.Seconds()
	}

	log.Printf(
		"📊 Download Metrics: %d files, %.2f MB, %.2f MB/s, %s provider, %.1fs duration",
		m.FilesCount, float64(m.FileSize)/(1024*1024), m.AvgSpeed, m.Provider, m.Duration.Seconds(),
	)
}

type DownloadOptions struct {
	AudioOnly  bool
	UseCookies bool
	WithBot    bool
	BotAPI     interface{}
	ChatID     int64
	MessageID  int
	Username   string
	Timeout    time.Duration
}

type DownloadResult struct {
	FilePaths []string
	TotalSize int64
	Provider  string
	Metrics   *DownloadMetrics
	Error     error
}

type FileInfo struct {
	Path        string
	Size        int64
	ContentType string
	Extension   string
	Checksum    string
}
