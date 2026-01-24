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
	return &Manager{
		config:   cfg,
		state:    NewStateManager(),
		resource: NewResourceManager(cfg.MaxConcurrentStreams),
	}
}

func (m *Manager) Stream(ctx context.Context, input StreamInput, uploadFn func(context.Context, Chunk, int64) error, progressFn func(int64, int64)) (int, error) {
	// 1. Acquire Resource
	if err := m.resource.Acquire(ctx); err != nil {
		return 0, fmt.Errorf("resource acquire failed: %w", err)
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
	parts, err := pipeline.Start(ctx, input, state)
	
	if err != nil {
		logger.Error("Stream failed", "error", err)
		return parts, err
	}

	state.mu.Lock()
	state.IsCompleted = true
	state.mu.Unlock()

	logger.Info("Stream completed", "file", input.Filename, "parts", parts)
	return parts, nil
}
