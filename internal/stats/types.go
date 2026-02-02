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
	Load1    float64
	Load5    float64
	Load15   float64

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
	StackInUse uint64
	NextGC     uint64
	PauseTotal uint64
	GCRuns     uint32
}

type BotStats struct {
	mu        sync.RWMutex
	StartTime time.Time

	Downloads        int64
	UniqueUsers      map[int64]bool
	LastDownloadTime time.Time

	NetSentBaseline uint64
	NetRecvBaseline uint64
}
