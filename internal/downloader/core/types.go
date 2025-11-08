package core

import (
	"fmt"
	"log"
	"time"
)

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

func ConvertToBytes(size float64, unit string) float64 {
	if multiplier, ok := UnitMultipliers[unit]; ok {
		return size * multiplier
	}
	return size
}

func FormatBytes(bytes float64) string {
	if bytes < 0 {
		return "0 B"
	}
	for _, unit := range SizeUnits {
		if bytes >= unit.Multiplier {
			value := bytes / unit.Multiplier
			return fmt.Sprintf("%.2f %s", value, unit.Name)
		}
	}
	return fmt.Sprintf("%.2f B", bytes)
}
