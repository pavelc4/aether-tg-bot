package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shirou/gopsutil/v4/cpu"
)

// CPUBasedConnectionManager manages aria2c connections based on CPU load
type CPUBasedConnectionManager struct {
	minConnections int
	maxConnections int
	updateInterval time.Duration
	currentCPU     float64
	mu             sync.RWMutex
	enabled        bool
}

var (
	cpuManager     *CPUBasedConnectionManager
	cpuManagerOnce sync.Once
)

// GetCPUManager returns singleton CPU manager instance
func GetCPUManager() *CPUBasedConnectionManager {
	cpuManagerOnce.Do(func() {
		cpuManager = &CPUBasedConnectionManager{
			minConnections: 4,  // Minimum connections saat CPU tinggi
			maxConnections: 16, // Maximum connections saat CPU rendah (hardcoded limit aria2c)
			updateInterval: 2 * time.Second,
			enabled:        os.Getenv("ADAPTIVE_ARIA2") != "false", // Default enabled
		}
		log.Printf("🔧 CPU Manager initialized (enabled=%v, min=%d, max=%d)",
			cpuManager.enabled, cpuManager.minConnections, cpuManager.maxConnections)
	})
	return cpuManager
}

// GetOptimalConnections returns optimal connection count based on CPU usage
func (m *CPUBasedConnectionManager) GetOptimalConnections(ctx context.Context) int {
	if !m.enabled {
		log.Println("⚙️  Adaptive aria2c disabled, using default 8 connections")
		return 8
	}

	// Get CPU usage percentage (averaged over 1 second)
	percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil || len(percentages) == 0 {
		log.Printf("⚠️  Failed to get CPU usage, using default: %v", err)
		return 8 // Default fallback
	}

	m.mu.Lock()
	m.currentCPU = percentages[0]
	m.mu.Unlock()

	// Calculate connections based on inverse CPU usage
	// Strategy: High CPU = fewer connections, Low CPU = more connections
	connections := m.calculateConnections(m.currentCPU)

	log.Printf("📊 CPU: %.2f%% → Using %d aria2c connections", m.currentCPU, connections)
	return connections
}

// calculateConnections maps CPU percentage to connection count
func (m *CPUBasedConnectionManager) calculateConnections(cpuPercent float64) int {
	switch {
	case cpuPercent > 85:
		return m.minConnections // 4 connections - CPU sangat tinggi
	case cpuPercent > 70:
		return 6 // CPU tinggi
	case cpuPercent > 55:
		return 8 // CPU menengah-tinggi
	case cpuPercent > 40:
		return 10 // CPU menengah
	case cpuPercent > 30:
		return 12 // CPU rendah-menengah
	case cpuPercent > 20:
		return 14 // CPU rendah
	default:
		return m.maxConnections // 16 connections - CPU sangat rendah
	}
}

// GetCurrentCPU returns current CPU usage (thread-safe)
func (m *CPUBasedConnectionManager) GetCurrentCPU() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCPU
}

// IsEnabled returns whether adaptive mode is enabled
func (m *CPUBasedConnectionManager) IsEnabled() bool {
	return m.enabled
}

// SetEnabled enables or disables adaptive mode
func (m *CPUBasedConnectionManager) SetEnabled(enabled bool) {
	m.enabled = enabled
	log.Printf("🔧 Adaptive aria2c %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

// MonitorCPUDuringDownload monitors CPU usage during download process
func (m *CPUBasedConnectionManager) MonitorCPUDuringDownload(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	if !m.enabled {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 CPU monitoring stopped")
			return
		case <-ticker.C:
			percentages, err := cpu.PercentWithContext(ctx, time.Second, false)
			if err == nil && len(percentages) > 0 {
				m.mu.Lock()
				m.currentCPU = percentages[0]
				m.mu.Unlock()

				// Log warning jika CPU terlalu tinggi
				if m.currentCPU > 90 {
					log.Printf("⚠️  HIGH CPU WARNING: %.2f%%", m.currentCPU)
				} else if m.currentCPU > 75 {
					log.Printf("⚡ CPU load: %.2f%%", m.currentCPU)
				} else {
					log.Printf("✅ CPU load: %.2f%%", m.currentCPU)
				}

				// Optional: Update user message dengan CPU info
				if bot != nil && chatID != 0 && msgID != 0 {
					// Bisa ditambahkan update ke user di sini jika diperlukan
				}
			}
		}
	}
}

// GetCPUStats returns formatted CPU statistics
func (m *CPUBasedConnectionManager) GetCPUStats() string {
	cpu := m.GetCurrentCPU()
	connections := m.calculateConnections(cpu)
	status := "Normal"

	if cpu > 85 {
		status = "Critical"
	} else if cpu > 70 {
		status = "High"
	} else if cpu > 40 {
		status = "Medium"
	} else {
		status = "Low"
	}

	return fmt.Sprintf("CPU: %.1f%% (%s) | Connections: %d", cpu, status, connections)
}

// BuildAria2Args builds aria2c arguments with adaptive connections
func (m *CPUBasedConnectionManager) BuildAria2Args(ctx context.Context) string {
	if !isAria2Available() {
		return ""
	}

	connections := m.GetOptimalConnections(ctx)

	// Build aria2c arguments
	// -c: continue download
	// -x: max connections per server
	// -s: split download
	// -k: min split size
	// --file-allocation=none: don't pre-allocate file space (faster start)
	// --max-tries: retry count
	// --retry-wait: wait time between retries
	args := fmt.Sprintf(
		"-c -x %d -s %d -k 1M --file-allocation=none --max-tries=5 --retry-wait=3",
		connections, connections,
	)

	log.Printf("🔗 Aria2c args: %s", args)
	return args
}

// isAria2Available checks if aria2c is available in PATH
func isAria2Available() bool {
	if os.Getenv("USE_ARIA2") == "false" {
		return false
	}
	_, err := exec.LookPath("aria2c")
	return err == nil
}
