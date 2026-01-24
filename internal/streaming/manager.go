package streaming

import (
	"context"
	"fmt"
	"time"

	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type Manager struct {
	config   Config
	state    *StateManager
	resource *ResourceManager
	// bufferPool *buffer.Pool // Could use a custom pool here if needed
}



func NewManager(cfg Config) *Manager {
	// Ensure global buffer pool matches chunk size or create local pool
	// For now, we assume global pool is used, but we might ideally resize it.
	// Since buffer package has a global Default, we can't easily resize it globally without side effects.
	// But `pipeline.go` uses `buffer.Get()`.
	// Correct approach: Update `pkg/buffer` to allow custom pools or `Recycle`.
	// For this implementation, we will proceed assuming 512KB chunks OR update buffer pool.
	// Let's stick to 512KB buffers if using default pool, or update Config.ChunkSize to 512KB.
	// User requested 1MB? The config default is 1MB.
	// So we should probably update `pkg/buffer/pool.go` or use `make` in implementation if size mismatch.
	
	return &Manager{
		config:   cfg,
		state:    NewStateManager(),
		resource: NewResourceManager(cfg.MaxConcurrentStreams),
	}
}

func (m *Manager) Stream(ctx context.Context, input StreamInput, uploadFn func(context.Context, Chunk, int64) error, progressFn func(int64, int64)) (int, string, error) {
	// 1. Acquire Resource
	if err := m.resource.Acquire(ctx); err != nil {
		return 0, "", fmt.Errorf("resource acquire failed: %w", err)
	}
	defer m.resource.Release()

	// 2. Create Transfer ID (simple unique string)
	streamID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), input.Size)
	
	// 3. Initialize State
	fileID := time.Now().UnixNano()
	state := m.state.NewState(streamID, fileID, input.Size)
	defer m.state.DeleteState(streamID)

	logger.Info("Starting stream", "file", input.Filename, "url", input.URL)

	// 4. Start Pipeline
	pipeline := NewPipeline(m.config, uploadFn, progressFn)
	parts, md5sum, err := pipeline.Start(ctx, input, state)
	
	if err != nil {
		logger.Error("Stream failed", "error", err)
		return parts, "", err
	}

	state.mu.Lock()
	state.IsCompleted = true
	state.mu.Unlock()

	logger.Info("Stream completed", "file", input.Filename, "parts", parts)
	return parts, md5sum, nil
}
