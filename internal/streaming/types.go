package streaming

import (
	"context"
	"sync"
)

type Config struct {
	MaxConcurrentStreams int
	UploadWorkers        int
	BufferSize           int
	ChunkSize            int64
	RetryLimit           int
}

type Chunk struct {
	PartNum    int    // 0-indexed part number
	TotalParts int    // Total number of parts
	Data       []byte // The actual data
	Size       int    // Size of data
}

type StreamState struct {
	mu            sync.Mutex
	FileID        int64           // Telegram File ID (generated)
	TotalParts    int             // Estimated total parts
	TotalSize     int64           // Total file size
	UploadedParts map[int]bool    // Map of uploaded parts
	ChunkRetries  map[int]int     // Retry count per part
	IsCompleted   bool            // Upload completed
}

type StreamInput struct {
	URL      string
	Filename string
	Size     int64
	Headers  map[string]string
	MIME     string
}

// Pipeline components
type StreamFunc func(ctx context.Context, input StreamInput) error
