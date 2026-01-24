package telegram

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gotd/td/tg"
)

type ProgressTracker struct {
	api       *tg.Client
	peer      tg.InputPeerClass
	msgID     int
	lastTime  time.Time
	mu        sync.Mutex
	minPeriod time.Duration
}

func NewProgressTracker(api *tg.Client, peer tg.InputPeerClass, msgID int) *ProgressTracker {
	return &ProgressTracker{
		api:       api,
		peer:      peer,
		msgID:     msgID,
		minPeriod: 2 * time.Second, 
	}
}

func (pt *ProgressTracker) Update(uploadedBytes, totalBytes int64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	if now.Sub(pt.lastTime) < pt.minPeriod && totalBytes > 0 && uploadedBytes < totalBytes {
		return
	}

	percent := float64(0)
	if totalBytes > 0 {
		percent = float64(uploadedBytes) / float64(totalBytes) * 100
	}

	text := fmt.Sprintf("⬇️ Downloading & Uploading...\nProgress: %.2f%% (%s / %s)", 
		percent, formatBytes(uploadedBytes), formatBytes(totalBytes))

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		_, err := pt.api.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
			Peer:    pt.peer,
			ID:      pt.msgID,
			Message: text,
		})
		if err != nil {
			log.Printf("Progress update failed: %v", err)
		}
	}()

	pt.lastTime = now
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
