package ui

import (
	"log"
	"time"
)

func NewUploadTracker(chatID int64, msgID int, filename string, totalSize int64, username string) *UploadTracker {
	return &UploadTracker{
		TotalSize:  totalSize,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		Filename:   filename,
	}
}

func (ut *UploadTracker) Update(current int64) {
	ut.mu.Lock()
	defer ut.mu.Unlock()

	now := time.Now()
	if now.Sub(ut.LastUpdate) < 3*time.Second {
		return
	}

	percentage := float64(current) / float64(ut.TotalSize) * 100
	log.Printf("Upload Progress [%s]: %.2f%%", ut.Filename, percentage)
	ut.LastUpdate = now
}

func (ut *UploadTracker) Complete() {
	log.Printf("Upload Complete: %s", ut.Filename)
}
