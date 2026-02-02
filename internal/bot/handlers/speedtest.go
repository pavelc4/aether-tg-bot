package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/tg"
)

func (h *Handler) handleSpeedtest(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	peer, err := resolvePeer(entities, msg.PeerID)
	if err != nil {
		return err
	}

	updates, err := h.Sender.To(peer).Text(ctx, "Testing connection...")
	if err != nil {
		return err
	}

	msgID, hasMsgID := getMsgID(updates)

	result := utils.RunSpeedTest()

	if result.Error != nil {
		errMsg := fmt.Sprintf("Speedtest failed: %v", result.Error)
		if hasMsgID {
			h.Client.API().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
				Peer:    peer,
				ID:      msgID,
				Message: errMsg,
			})
		} else {
			h.Sender.To(peer).Text(ctx, errMsg)
		}
		return nil
	}

	// Final Report
	finalText := fmt.Sprintf(
		"Cloudflare Speedtest\n\n"+
			"Server: %s\n"+
			"Your IP: %s\n"+
			"Latency: %s\n\n"+
			"Download: %.2f Mbps\n"+
			"Upload: %.2f Mbps",
		result.ServerLocation,
		result.ClientIP,
		result.Latency.Round(time.Millisecond),
		result.DownloadSpeed,
		result.UploadSpeed,
	)

	if hasMsgID {
		h.Client.API().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
			Peer:    peer,
			ID:      msgID,
			Message: finalText,
		})
	} else {
		h.Sender.To(peer).Text(ctx, finalText)
	}

	return nil
}
