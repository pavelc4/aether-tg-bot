package stats

import (
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

var (
	globalStats *BotStats
	once        sync.Once
)

func GetStats() *BotStats {
	once.Do(func() {
		globalStats = &BotStats{
			StartTime:   time.Now(),
			UniqueUsers: make(map[int64]bool),
		}

		if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
			globalStats.NetSentBaseline = netStats[0].BytesSent
			globalStats.NetRecvBaseline = netStats[0].BytesRecv
		}
	})
	return globalStats
}

func TrackUser(userID int64) {
	s := GetStats()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UniqueUsers[userID] = true
}

func TrackDownload() {
	s := GetStats()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Downloads++
	s.LastDownloadTime = time.Now()
}

func GetUptime() string {
	return time.Since(GetStats().StartTime).Round(time.Second).String()
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
	if cores, err := cpu.Counts(false); err == nil {
		info.CPUCores = cores
	} else {
		info.CPUCores = runtime.NumCPU()
	}

	if avg, err := load.Avg(); err == nil {
		info.Load1 = avg.Load1
		info.Load5 = avg.Load5
		info.Load15 = avg.Load15
	}

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
	info.StackInUse = m.StackInuse
	info.NextGC = m.NextGC
	info.PauseTotal = m.PauseTotalNs
	info.GCRuns = m.NumGC

	return info, nil
}
