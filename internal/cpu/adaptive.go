package cpu

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gocpu "github.com/shirou/gopsutil/v4/cpu"
)

type CPUBasedConnectionManager struct {
	minConnections int
	maxConnections int
	updateInterval time.Duration
	currentCPU     float64
	mu             sync.RWMutex
	enabled        bool
}

func NewAdaptiveManager() *CPUBasedConnectionManager {
	manager := &CPUBasedConnectionManager{
		minConnections: 4,
		maxConnections: 16,
		updateInterval: 2 * time.Second,
		enabled:        os.Getenv("ADAPTIVE_ARIA2") != "false",
	}

	log.Printf("Adaptive CPU Manager initialized (enabled=%v, min=%d, max=%d)",
		manager.enabled, manager.minConnections, manager.maxConnections)

	return manager
}

func (m *CPUBasedConnectionManager) IsEnabled() bool {
	return m.enabled
}

func (m *CPUBasedConnectionManager) GetOptimalConnections(ctx context.Context) int {
	if !m.enabled {
		log.Println("Adaptive aria2c disabled, using default 8 connections")
		return 8
	}

	percentages, err := gocpu.PercentWithContext(ctx, time.Second, false)
	if err != nil || len(percentages) == 0 {
		log.Printf("Failed to get CPU usage, using default: %v", err)
		return 8
	}

	m.mu.Lock()
	m.currentCPU = percentages[0]
	m.mu.Unlock()

	connections := m.calculateConnections(m.currentCPU)
	log.Printf(" CPU: %.2f%% → Using %d aria2c connections", m.currentCPU, connections)
	return connections
}

type cpuLevels map[float64]int

func (m *CPUBasedConnectionManager) levels() cpuLevels {
	return cpuLevels{
		85: m.minConnections,
		70: 6,
		55: 8,
		40: 10,
		30: 12,
		20: 14,
	}
}

func (m *CPUBasedConnectionManager) calculateConnections(cpu float64) int {
	levels := m.levels()
	order := []float64{85, 70, 55, 40, 30, 20}

	for _, t := range order {
		if cpu > t {
			return levels[t]
		}
	}
	return m.maxConnections
}

func (m *CPUBasedConnectionManager) GetCurrentCPU() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCPU
}

func (m *CPUBasedConnectionManager) SetEnabled(enabled bool) {
	m.enabled = enabled
	status := map[bool]string{true: "enabled", false: "disabled"}[enabled]
	log.Printf("Adaptive aria2c %s", status)
}

func (m *CPUBasedConnectionManager) MonitorCPUDuringDownload(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, msgID int) {
	if !m.enabled {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("CPU monitoring stopped")
			return
		case <-ticker.C:
			percentages, err := gocpu.PercentWithContext(ctx, time.Second, false)
			if err == nil && len(percentages) > 0 {
				m.mu.Lock()
				m.currentCPU = percentages[0]
				m.mu.Unlock()

				if m.currentCPU > 90 {
					log.Printf(" HIGH CPU WARNING: %.2f%%", m.currentCPU)
				} else if m.currentCPU > 75 {
					log.Printf(" CPU load: %.2f%%", m.currentCPU)
				} else {
					log.Printf(" CPU load: %.2f%%", m.currentCPU)
				}
			}
		}
	}
}

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

func (m *CPUBasedConnectionManager) BuildAria2Args(ctx context.Context) string {
	if !isAria2Available() {
		return ""
	}

	connections := m.GetOptimalConnections(ctx)
	args := fmt.Sprintf(
		"-c -x %d -s %d -k 1M --file-allocation=none --max-tries=5 --retry-wait=3",
		connections, connections,
	)

	log.Printf("Aria2c args: %s", args)
	return args
}

func isAria2Available() bool {
	if os.Getenv("USE_ARIA2") == "false" {
		return false
	}
	_, err := exec.LookPath("aria2c")
	return err == nil
}
