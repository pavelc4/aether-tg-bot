package handler

import (
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/cache"
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

func getMsgID(updates tg.UpdatesClass) int {
	switch u := updates.(type) {
	case *tg.UpdateShortSentMessage:
		return u.ID
	case *tg.Updates:
		for _, update := range u.Updates {
			if msg, ok := update.(*tg.UpdateNewMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID
				}
			}
			if msg, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID
				}
			}
		}
	}
	return 0
}

func getMediaFromUpdates(updates tg.UpdatesClass) *cache.CachedMedia {
	var msg *tg.Message
	
	switch u := updates.(type) {
	case *tg.UpdateShortSentMessage:
		return nil
	case *tg.Updates:
		for _, update := range u.Updates {
			if m, ok := update.(*tg.UpdateNewMessage); ok {
				if mm, ok := m.Message.(*tg.Message); ok {
					msg = mm
					break
				}
			}
			if m, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if mm, ok := m.Message.(*tg.Message); ok {
					msg = mm
					break
				}
			}
		}
	}

	if msg == nil {
		return nil
	}

	switch m := msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		if photo, ok := m.Photo.(*tg.Photo); ok {
			return &cache.CachedMedia{
				ID:            photo.ID,
				AccessHash:    photo.AccessHash,
				FileReference: photo.FileReference,
				Type:          cache.TypePhoto,
			}
		}
	case *tg.MessageMediaDocument:
		if doc, ok := m.Document.(*tg.Document); ok {
			return &cache.CachedMedia{
				ID:            doc.ID,
				AccessHash:    doc.AccessHash,
				FileReference: doc.FileReference,
				Type:          cache.TypeDocument,
			}
		}
	}
	
	return nil
}
