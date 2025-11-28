package cpu

import (
	"context"
	"log"
)

// CPUManager interface
type CPUManager interface {
	IsEnabled() bool
	BuildAria2Args(ctx context.Context) string
	GetOptimalConnections(ctx context.Context) int
}

// Global CPU manager
var cpuManager CPUManager

// GetCPUManager return current CPU manager
func GetCPUManager() CPUManager {
	if cpuManager == nil {
		// Initialize default if not set
		cpuManager = NewAdaptiveManager()
	}
	return cpuManager
}

// SetCPUManager set custom CPU manager
func SetCPUManager(manager CPUManager) {
	if manager != nil {
		cpuManager = manager
		log.Printf("CPU Manager set: %T", manager)
	}
}

// SimpleCPUManager simple implementation
type SimpleCPUManager struct {
	enabled bool
}

func NewSimpleCPUManager() *SimpleCPUManager {
	return &SimpleCPUManager{enabled: false}
}

func (m *SimpleCPUManager) IsEnabled() bool {
	return m.enabled
}

func (m *SimpleCPUManager) BuildAria2Args(ctx context.Context) string {
	return "-c -x 16 -s 16 -k 1M --file-allocation=none"
}

func (m *SimpleCPUManager) GetOptimalConnections(ctx context.Context) int {
	return 16
}
