package ui

import (
	"sync"
	"time"
)

type DownloadProgress struct {
	Percentage float64
	Downloaded string
	Speed      string
	ETA        string
	Status     string
}

type UploadProgress struct {
	Percentage float64
	Uploaded   string
	TotalSize  string
	Speed      string
}
type UploadTracker struct {
	TotalSize  int64
	StartTime  time.Time
	LastUpdate time.Time
	Filename   string
	mu         sync.Mutex
}
