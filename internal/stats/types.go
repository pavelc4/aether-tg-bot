package stats

import (
	"sync"
	"time"
)

type SystemInfo struct {
	OS           string
	Hostname     string
	SystemUptime time.Duration

	CPUCores int
	CPUUsage float64

	MemUsed      uint64
	MemTotal     uint64
	MemPercent   float64
	MemAvailable uint64

	DiskUsed    uint64
	DiskTotal   uint64
	DiskPercent float64
	DiskFree    uint64

	NetSent uint64
	NetRecv uint64

	ProcessPID    int
	ProcessUptime time.Duration
	ProcessCPU    float64
	ProcessMem    uint64

	GoVersion  string
	Goroutines int
	HeapAlloc  uint64
	GCRuns     uint32
}

type BotStats struct {
	mu        sync.RWMutex
	StartTime time.Time

	Downloads        int64
	TotalFiles       int64
	TotalBytes       int64
	SuccessDownloads int64
	FailedDownloads  int64
	AudioDownloads   int64
	VideoDownloads   int64
	ImageDownloads   int64

	UniqueUsers map[int64]bool

	PlatformStats map[string]int64

	DailyStats   map[string]*PeriodStats // YYYY-MM-DD
	WeeklyStats  map[string]*PeriodStats // YYYY-Www
	MonthlyStats map[string]*PeriodStats // YYYY-MM

	LastDownloadTime time.Time

	NetSentBaseline uint64
	NetRecvBaseline uint64
}

type PeriodStats struct {
	Downloads int64
	Files     int64
	Bytes     int64
	Users     map[int64]bool
}
