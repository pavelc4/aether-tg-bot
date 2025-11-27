package stats

import (
	"encoding/json"
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

type BotStats struct {
	mu        sync.RWMutex
	StartTime time.Time

	TotalDownloads   int64
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

		if err := globalStats.LoadFromFile(); err != nil {
			fmt.Printf("Failed to load stats: %v\n", err)
		} else {
			fmt.Printf("Stats loaded: %d downloads\n", globalStats.TotalDownloads)
		}

		globalStats.StartAutoSave(5 * time.Minute)

		if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
			globalStats.NetSentBaseline = netStats[0].BytesSent
			globalStats.NetRecvBaseline = netStats[0].BytesRecv
		}
	})
	return globalStats
}

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

	incrementers := map[string]func(){
		"Audio": func() { s.AudioDownloads++ },
		"Video": func() { s.VideoDownloads++ },
		"Image": func() { s.ImageDownloads++ },
	}

	if inc, ok := incrementers[mediaType]; ok {
		inc()
	}

	s.UniqueUsers[userID] = true

	if platform != "" && platform != "Unknown" {
		s.PlatformStats[platform]++
	}

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

func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	if hostInfo, err := host.Info(); err == nil {
		info.OS = hostInfo.OS
		info.Hostname = hostInfo.Hostname
		info.SystemUptime = time.Duration(hostInfo.Uptime) * time.Second
	}

	if cpuPercent, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercent) > 0 {
		info.CPUUsage = cpuPercent[0]
	}
	info.CPUCores = runtime.NumCPU()

	if memInfo, err := mem.VirtualMemory(); err == nil {
		info.MemUsed = memInfo.Used
		info.MemTotal = memInfo.Total
		info.MemPercent = memInfo.UsedPercent
		info.MemAvailable = memInfo.Available
	}

	if diskInfo, err := disk.Usage("/"); err == nil {
		info.DiskUsed = diskInfo.Used
		info.DiskTotal = diskInfo.Total
		info.DiskPercent = diskInfo.UsedPercent
		info.DiskFree = diskInfo.Free
	}

	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		stats := GetStats()
		info.NetSent = netStats[0].BytesSent - stats.NetSentBaseline
		info.NetRecv = netStats[0].BytesRecv - stats.NetRecvBaseline
	}

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

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info.GoVersion = runtime.Version()
	info.Goroutines = runtime.NumGoroutine()
	info.HeapAlloc = m.Alloc
	info.GCRuns = m.NumGC

	return info, nil
}

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

func (s *BotStats) GetPeriodStats(period string) *PeriodStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()

	strategies := map[string]func() *PeriodStats{
		"today": func() *PeriodStats { return s.DailyStats[now.Format("2006-01-02")] },
		"week":  func() *PeriodStats { return s.WeeklyStats[now.Format("2006-W")+getWeekNumber(now)] },
		"month": func() *PeriodStats { return s.MonthlyStats[now.Format("2006-01")] },
	}

	if strategy, exists := strategies[period]; exists {
		return strategy()
	}

	return nil
}

const statsFilePath = "/app/data/stats.json"

func (s *BotStats) SaveToFile() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := os.MkdirAll("/app/data", 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statsFilePath, data, 0644)
}

func (s *BotStats) LoadFromFile() error {
	data, err := os.ReadFile(statsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return json.Unmarshal(data, s)
}

func (s *BotStats) StartAutoSave(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := s.SaveToFile(); err != nil {
				fmt.Printf("Failed to save stats: %v\n", err)
			}
		}
	}()
}
