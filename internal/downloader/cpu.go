package downloader

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CPUManager interface untuk adaptive downloads
type CPUManager interface {
	IsEnabled() bool
	BuildAria2Args(ctx context.Context) string
	GetOptimalConnections(ctx context.Context) int
	MonitorCPUDuringDownload(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, msgID int)
}

// SimpleCPUManager is a simple implementation
type SimpleCPUManager struct {
	enabled bool
}

var cpuManager CPUManager = &SimpleCPUManager{enabled: false}

// GetCPUManager returns global CPU manager instance
func GetCPUManager() CPUManager {
	return cpuManager
}

// SetCPUManager sets custom CPU manager
func SetCPUManager(manager CPUManager) {
	cpuManager = manager
}

// IsEnabled returns whether adaptive download is enabled
func (m *SimpleCPUManager) IsEnabled() bool {
	return m.enabled
}

// BuildAria2Args returns aria2c arguments
func (m *SimpleCPUManager) BuildAria2Args(ctx context.Context) string {
	return "-c -x 16 -s 16 -k 1M --file-allocation=none"
}

// GetOptimalConnections returns optimal connection count
func (m *SimpleCPUManager) GetOptimalConnections(ctx context.Context) int {
	return 16
}

// MonitorCPUDuringDownload monitors CPU during download (stub)
func (m *SimpleCPUManager) MonitorCPUDuringDownload(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, msgID int) {
}
