package handlers

import (
	"github.com/pavelc4/aether-tg-bot/config"
)

func isOwner(userID int64) bool {
	ownerID := config.GetOwnerID()
	if ownerID == 0 {
		return false
	}
	return userID == ownerID
}

func requireOwner(userID int64) bool {
	return isOwner(userID)
}
