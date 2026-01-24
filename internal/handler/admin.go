package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
)

type AdminHandler struct {
	client *telegram.Client
}

func NewAdminHandler(cli *telegram.Client) *AdminHandler {
	return &AdminHandler{client: cli}
}

func (h *AdminHandler) HandleStats(ctx context.Context, e tg.Entities, msg *tg.Message) error {
	senderID := getSenderID(msg)
	if senderID != config.GetOwnerID() {
		return nil // Ignore non-owner
	}

	sysInfo, err := stats.GetSystemInfo()
	if err != nil {
		sender := message.NewSender(h.client.API())
		inputPeer, _ := resolvePeer(msg.PeerID, e)
		sender.To(inputPeer).Reply(msg.ID).Text(ctx, fmt.Sprintf("❌ Failed to get stats: %v", err))
		return err
	}

	text := fmt.Sprintf(
		"<b>CPU</b>\n"+
			"├─ Cores: <code>%d</code>\n"+
			"└─ Usage: <code>%.2f%%</code>\n"+
			"└─ Uptime: <code>%v</code>\n\n"+
			"<b>Memory</b>\n"+
			"├─ Used: <code>%s / %s (%.1f%%)</code>\n"+
			"└─ Available: <code>%s</code>\n\n"+
			"<b>Network</b>\n"+
			"├─ Sent: <code>%s</code>\n"+
			"└─ Received: <code>%s</code>\n\n"+
			"<b>Bot Process</b>\n"+
			"├─ Uptime: <code>%v</code>\n"+
			"├─ PID: <code>%d</code>\n"+
			"├─ CPU: <code>%.2f%%</code>\n"+
			"├─ Memory: <code>%s</code>\n"+
			"└─ Go Version: <code>%s</code>\n\n"+
			"<b>Go Runtime</b>\n"+
			"├─ Goroutines: <code>%d</code>\n"+
			"├─ Heap Alloc: <code>%s</code>\n"+
			"└─ GC Runs: <code>%d</code>",
		sysInfo.CPUCores,
		sysInfo.CPUUsage,
		sysInfo.SystemUptime.Round(time.Second),
		formatBytes(sysInfo.MemUsed), formatBytes(sysInfo.MemTotal), sysInfo.MemPercent,
		formatBytes(sysInfo.MemAvailable),
		formatBytes(sysInfo.NetSent),
		formatBytes(sysInfo.NetRecv),
		sysInfo.ProcessUptime.Round(time.Second),
		sysInfo.ProcessPID,
		sysInfo.ProcessCPU,
		formatBytes(sysInfo.ProcessMem),
		sysInfo.GoVersion,
		sysInfo.Goroutines,
		formatBytes(sysInfo.HeapAlloc),
		sysInfo.GCRuns,
	)

	sender := message.NewSender(h.client.API())
	inputPeer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}
	_, err = sender.To(inputPeer).Reply(msg.ID).StyledText(ctx, html.String(nil, text))
	return err
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	val := int64(b)
	for n := val / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(val)/float64(div), "KMGTPE"[exp])
}

func getSenderID(msg *tg.Message) int64 {
	if peer, ok := msg.PeerID.(*tg.PeerUser); ok {
		return peer.UserID
	}
	return 0
}
