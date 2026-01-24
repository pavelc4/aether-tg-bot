package stats

import (
	"sync"
	"time"
)

type Stats struct {
	mu            sync.RWMutex
	Downloads     int64
	TotalBytes    int64
	TotalDuration time.Duration
	StartTime     time.Time
}

var globalStats = &Stats{
	StartTime: time.Now(),
}

func RecordDownload(size int64, duration time.Duration) {
	globalStats.mu.Lock()
	defer globalStats.mu.Unlock()

	globalStats.Downloads++
	globalStats.TotalBytes += size
	globalStats.TotalDuration += duration
}

type StatSnapshot struct {
	Downloads   int64
	TotalBytes  int64
	AvgDuration time.Duration
	Uptime      time.Duration
}

func GetSnapshot() StatSnapshot {
	globalStats.mu.RLock()
	defer globalStats.mu.RUnlock()

	avg := time.Duration(0)
	if globalStats.Downloads > 0 {
		avg = globalStats.TotalDuration / time.Duration(globalStats.Downloads)
	}

	return StatSnapshot{
		Downloads:   globalStats.Downloads,
		TotalBytes:  globalStats.TotalBytes,
		AvgDuration: avg,
		Uptime:      time.Since(globalStats.StartTime),
	}
}
