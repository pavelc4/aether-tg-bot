package stats

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	globalStats *BotStats
	once        sync.Once
)

// BotStats tracks bot statistics
type BotStats struct {
	mu        sync.RWMutex
	StartTime time.Time

	// Total stats
	TotalDownloads   int64
	TotalFiles       int64
	TotalBytes       int64
	SuccessDownloads int64
	FailedDownloads  int64
	AudioDownloads   int64
	VideoDownloads   int64
	ImageDownloads   int64

	// User tracking
	UniqueUsers map[int64]bool

	// Platform stats
	PlatformStats map[string]int64

	// Time-based stats
	DailyStats   map[string]*PeriodStats // YYYY-MM-DD
	WeeklyStats  map[string]*PeriodStats // YYYY-Www
	MonthlyStats map[string]*PeriodStats // YYYY-MM

	LastDownloadTime time.Time

	// Network stats baseline
	NetSentBaseline uint64
	NetRecvBaseline uint64
}

// PeriodStats tracks stats for a specific period
type PeriodStats struct {
	Downloads int64
	Files     int64
	Bytes     int64
	Users     map[int64]bool
}

// GetStats returns global stats instance
func GetStats() *BotStats {
	once.Do(func() {
		globalStats = &BotStats{
			StartTime:     time.Now(),
			UniqueUsers:   make(map[int64]bool),
			PlatformStats: make(map[string]int64),
			DailyStats:    make(map[string]*PeriodStats),
			WeeklyStats:   make(map[string]*PeriodStats),
			MonthlyStats:  make(map[string]*PeriodStats),
		}

		// Initialize network baseline
		if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
			globalStats.NetSentBaseline = netStats[0].BytesSent
			globalStats.NetRecvBaseline = netStats[0].BytesRecv
		}
	})
	return globalStats
}

// RecordDownload records a download
func (s *BotStats) RecordDownload(userID int64, platform, mediaType string, files int, bytes int64, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	s.TotalDownloads++
	s.TotalFiles += int64(files)
	s.TotalBytes += bytes
	s.LastDownloadTime = now

	if success {
		s.SuccessDownloads++
	} else {
		s.FailedDownloads++
	}

	// Track media type
	switch mediaType {
	case "Audio":
		s.AudioDownloads++
	case "Video":
		s.VideoDownloads++
	case "Image":
		s.ImageDownloads++
	}

	// Track unique users
	s.UniqueUsers[userID] = true

	// Track platform
	if platform != "" && platform != "Unknown" {
		s.PlatformStats[platform]++
	}

	// Track time-based stats
	dayKey := now.Format("2006-01-02")
	weekKey := now.Format("2006-W") + getWeekNumber(now)
	monthKey := now.Format("2006-01")

	s.recordPeriodStats(s.DailyStats, dayKey, userID, files, bytes)
	s.recordPeriodStats(s.WeeklyStats, weekKey, userID, files, bytes)
	s.recordPeriodStats(s.MonthlyStats, monthKey, userID, files, bytes)
}

func (s *BotStats) recordPeriodStats(stats map[string]*PeriodStats, key string, userID int64, files int, bytes int64) {
	if stats[key] == nil {
		stats[key] = &PeriodStats{
			Users: make(map[int64]bool),
		}
	}

	stats[key].Downloads++
	stats[key].Files += int64(files)
	stats[key].Bytes += bytes
	stats[key].Users[userID] = true
}

func getWeekNumber(t time.Time) string {
	_, week := t.ISOWeek()
	return fmt.Sprintf("%02d", week)
}

// GetSystemInfo returns complete system information
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Host info
	if hostInfo, err := host.Info(); err == nil {
		info.OS = hostInfo.OS
		info.Hostname = hostInfo.Hostname
		info.SystemUptime = time.Duration(hostInfo.Uptime) * time.Second
	}

	// CPU info
	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		info.CPUUsage = cpuPercent[0]
	}
	info.CPUCores = runtime.NumCPU()

	// Memory info
	if memInfo, err := mem.VirtualMemory(); err == nil {
		info.MemUsed = memInfo.Used
		info.MemTotal = memInfo.Total
		info.MemPercent = memInfo.UsedPercent
		info.MemAvailable = memInfo.Available
	}

	// Disk info
	if diskInfo, err := disk.Usage("/"); err == nil {
		info.DiskUsed = diskInfo.Used
		info.DiskTotal = diskInfo.Total
		info.DiskPercent = diskInfo.UsedPercent
		info.DiskFree = diskInfo.Free
	}

	// Network info
	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		stats := GetStats()
		info.NetSent = netStats[0].BytesSent - stats.NetSentBaseline
		info.NetRecv = netStats[0].BytesRecv - stats.NetRecvBaseline
	}

	// Process info
	if proc, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if cpuPercent, err := proc.CPUPercent(); err == nil {
			info.ProcessCPU = cpuPercent
		}
		if memInfo, err := proc.MemoryInfo(); err == nil {
			info.ProcessMem = memInfo.RSS
		}
	}

	info.ProcessPID = os.Getpid()
	info.ProcessUptime = time.Since(GetStats().StartTime)

	// Go runtime
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info.GoVersion = runtime.Version()
	info.Goroutines = runtime.NumGoroutine()
	info.HeapAlloc = m.Alloc
	info.GCRuns = m.NumGC

	return info, nil
}

// SystemInfo contains complete system information
type SystemInfo struct {
	// System
	OS           string
	Hostname     string
	SystemUptime time.Duration

	// CPU
	CPUCores int
	CPUUsage float64

	// Memory
	MemUsed      uint64
	MemTotal     uint64
	MemPercent   float64
	MemAvailable uint64

	// Disk
	DiskUsed    uint64
	DiskTotal   uint64
	DiskPercent float64
	DiskFree    uint64

	// Network
	NetSent uint64
	NetRecv uint64

	// Process
	ProcessPID    int
	ProcessUptime time.Duration
	ProcessCPU    float64
	ProcessMem    uint64

	// Go Runtime
	GoVersion  string
	Goroutines int
	HeapAlloc  uint64
	GCRuns     uint32
}

// GetPeriodStats returns stats for specific period
func (s *BotStats) GetPeriodStats(period string) *PeriodStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var key string

	switch period {
	case "today":
		key = now.Format("2006-01-02")
		return s.DailyStats[key]
	case "week":
		key = now.Format("2006-W") + getWeekNumber(now)
		return s.WeeklyStats[key]
	case "month":
		key = now.Format("2006-01")
		return s.MonthlyStats[key]
	}

	return nil
}
