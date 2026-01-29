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
	api          *tg.Client
	peer         tg.InputPeerClass
	msgID        int
	lastTime     time.Time
	lastBytes    int64
	startTime    time.Time
	mu           sync.Mutex
	minPeriod    time.Duration
	title        string
}

func NewProgressTracker(api *tg.Client, peer tg.InputPeerClass, msgID int) *ProgressTracker {
	return &ProgressTracker{
		api:       api,
		peer:      peer,
		msgID:     msgID,
		minPeriod: 4 * time.Second,
		startTime: time.Now(),
	}
}

func (pt *ProgressTracker) SetTitle(title string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.title = title
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

	speed := int64(0)
	if pt.lastBytes > 0 && !pt.lastTime.IsZero() {
		elapsed := now.Sub(pt.lastTime).Seconds()
		if elapsed > 0 {
			speed = int64(float64(uploadedBytes-pt.lastBytes) / elapsed)
		}
	}
	barWidth := 12
	filled := int(percent / 100 * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "■"
		} else {
			bar += "□"
		}
	}

	elapsed := time.Since(pt.startTime)
	
	title := pt.title
	if title == "" {
		title = "Download"
	}
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	text := fmt.Sprintf(
		"🎥 <b>%s</b>\n"+
		"┌ Status : <code>Downloading... (%.1f%%)</code>\n"+
		"├ [<code>%s</code>]\n"+
		"├ Size : <code>%s</code>\n"+
		"├ Processed : <code>%s</code>\n"+
		"├ Speed : <code>%s/s</code>\n"+
		"├ Time : <code>%s</code>\n"+
		"└ Engine : <code>yt-dlp</code>",
		title,
		percent,
		bar,
		formatBytes(uint64(totalBytes)),
		formatBytes(uint64(uploadedBytes)),
		formatBytes(uint64(speed)),
		formatDuration(elapsed),
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
	pt.lastBytes = uploadedBytes
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

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	
	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}
