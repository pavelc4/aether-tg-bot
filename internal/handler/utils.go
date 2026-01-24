package handler

import (
	"fmt"

	"github.com/gotd/td/tg"
)

// resolvePeer converts a PeerClass to InputPeerClass using the provided entities.
func resolvePeer(peer tg.PeerClass, entities tg.Entities) (tg.InputPeerClass, error) {
	switch p := peer.(type) {
	case *tg.PeerUser:
		user, ok := entities.Users[p.UserID]
		if !ok {
			return nil, fmt.Errorf("user %d not found in entities", p.UserID)
		}
		return &tg.InputPeerUser{
			UserID:     user.ID,
			AccessHash: user.AccessHash,
		}, nil
	case *tg.PeerChat:
		chat, ok := entities.Chats[p.ChatID]
		if !ok {
			return nil, fmt.Errorf("chat %d not found in entities", p.ChatID)
		}
		return &tg.InputPeerChat{
			ChatID: chat.ID,
		}, nil
	case *tg.PeerChannel:
		channel, ok := entities.Channels[p.ChannelID]
		if !ok {
			return nil, fmt.Errorf("channel %d not found in entities", p.ChannelID)
		}
		return &tg.InputPeerChannel{
			ChannelID:  channel.ID,
			AccessHash: channel.AccessHash,
		}, nil
	default:
		return nil, fmt.Errorf("unknown peer type: %T", peer)
	}
}
