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
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/internal/utils"
)

type AdminHandler struct {
	client    *telegram.Client
	streamMgr *streaming.Manager
}

func NewAdminHandler(cli *telegram.Client, sm *streaming.Manager) *AdminHandler {
	return &AdminHandler{client: cli, streamMgr: sm}
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
		"<b>System Status</b>\n\n"+
			"<b>OS Info</b>\n"+
			"├ System : <code>%s</code>\n"+
			"├ Host : <code>%s</code>\n"+
			"└ Uptime : <code>%s</code>\n\n"+
			"<b>CPU</b>\n"+
			"├ Cores : <code>%d</code>\n"+
			"├ Usage : <code>%.2f%%</code>\n"+
			"└ Load : <code>%.2f %.2f %.2f</code>\n\n"+
			"<b>Memory</b>\n"+
			"├ Used : <code>%s / %s (%.1f%%)</code>\n"+
			"└ Free : <code>%s</code>\n\n"+
			"<b>Network</b>\n"+
			"├ Sent : <code>%s</code>\n"+
			"└ Recv : <code>%s</code>\n\n"+
			"<b>Bot Process</b>\n"+
			"├ Uptime : <code>%s</code>\n"+
			"├ PID : <code>%d</code>\n"+
			"├ CPU : <code>%.2f%%</code>\n"+
			"├ Mem : <code>%s</code>\n"+
			"├ Pipes : <code>%d</code>\n"+
			"└ Go Ver : <code>%s</code>\n\n"+
			"<b>Go Process</b>\n"+
			"├ Routines : <code>%d</code>\n"+
			"├ Heap : <code>%s</code>\n"+
			"├ Stack : <code>%s</code>\n"+
			"├ Next GC : <code>%s</code>\n"+
			"├ GC Pause : <code>%s</code>\n"+
			"└ GC Runs : <code>%d</code>",
		sysInfo.OS,
		sysInfo.Hostname,
		sysInfo.SystemUptime.Round(time.Second),
		sysInfo.CPUCores,
		sysInfo.CPUUsage,
		sysInfo.Load1, sysInfo.Load5, sysInfo.Load15,
		utils.FormatBytes(sysInfo.MemUsed), utils.FormatBytes(sysInfo.MemTotal), sysInfo.MemPercent,
		utils.FormatBytes(sysInfo.MemAvailable),
		utils.FormatBytes(sysInfo.NetSent),
		utils.FormatBytes(sysInfo.NetRecv),
		sysInfo.ProcessUptime.Round(time.Second),
		sysInfo.ProcessPID,
		sysInfo.ProcessCPU,
		utils.FormatBytes(sysInfo.ProcessMem),
		h.streamMgr.GetActiveStreams(),
		sysInfo.GoVersion,
		sysInfo.Goroutines,
		utils.FormatBytes(sysInfo.HeapAlloc),
		utils.FormatBytes(sysInfo.StackInUse),
		utils.FormatBytes(sysInfo.NextGC),
		time.Duration(sysInfo.PauseTotal),
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



func getSenderID(msg *tg.Message) int64 {
	if from, ok := msg.GetFromID(); ok {
		if user, ok := from.(*tg.PeerUser); ok {
			return user.UserID
		}
	}
	if peer, ok := msg.PeerID.(*tg.PeerUser); ok {
		return peer.UserID
	}
	return 0
}
