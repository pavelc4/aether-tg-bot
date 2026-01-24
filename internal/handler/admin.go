package handler

import (
	"context"
	"fmt"
	"runtime"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
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

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	text := fmt.Sprintf(" System Stats\n\n"+
		"Goroutines: %d\n"+
		"Memory Alloc: %v MB\n"+
		"Memory Total: %v MB\n"+
		"Sys Memory: %v MB\n"+
		"Uptime: %v",
		runtime.NumGoroutine(),
		bToMb(m.Alloc),
		bToMb(m.TotalAlloc),
		bToMb(m.Sys),
		"TODO", // app uptime
	)

	sender := message.NewSender(h.client.API())
	inputPeer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}
	_, err = sender.To(inputPeer).Reply(msg.ID).Text(ctx, text)
	return err
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func getSenderID(msg *tg.Message) int64 {
	if peer, ok := msg.PeerID.(*tg.PeerUser); ok {
		return peer.UserID
	}
	return 0
}
