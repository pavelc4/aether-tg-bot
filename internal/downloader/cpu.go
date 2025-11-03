package downloader

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CPUManager interface {
	IsEnabled() bool
	BuildAria2Args(ctx context.Context) string
	GetOptimalConnections(ctx context.Context) int
	MonitorCPUDuringDownload(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, msgID int)
}

type SimpleCPUManager struct {
	enabled bool
}

var cpuManager CPUManager = &SimpleCPUManager{enabled: false}

func GetCPUManager() CPUManager {
	return cpuManager
}

func SetCPUManager(manager CPUManager) {
	cpuManager = manager
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

func (m *SimpleCPUManager) MonitorCPUDuringDownload(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, msgID int) {
}
