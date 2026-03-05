package telegram

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/utils"
)

type ProgressTracker struct {
	api       *tg.Client
	peer      tg.InputPeerClass
	msgID     int
	lastTime  time.Time
	startTime time.Time
	mu        sync.Mutex
	minPeriod time.Duration
	engine    string
}

func NewProgressTracker(api *tg.Client, peer tg.InputPeerClass, msgID int, engine string) *ProgressTracker {
	engineDisplay := convertEngineName(engine)
	return &ProgressTracker{
		api:       api,
		peer:      peer,
		msgID:     msgID,
		minPeriod: 5 * time.Second,
		startTime: time.Now(),
		engine:    engineDisplay,
	}
}

func convertEngineName(providerName string) string {
	switch providerName {
	case "TikTok":
		return "TikWM API"
	case "YouTube":
		return "yt-dlp"
	case "Cobalt":
		return "Cobalt API"
	default:
		if providerName != "" {
			return providerName
		}
		return "yt-dlp + Bun"
	}
}

func (pt *ProgressTracker) SetTitle(title string) {
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

	elapsed := time.Since(pt.startTime)

	text := fmt.Sprintf(
		"🎥 Uploading... %.1f%% | %s / %s | %s | Engine: %s",
		percent,
		utils.FormatBytes(uint64(uploadedBytes)),
		utils.FormatBytes(uint64(totalBytes)),
		utils.FormatDuration(elapsed),
		pt.engine,
	)

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
