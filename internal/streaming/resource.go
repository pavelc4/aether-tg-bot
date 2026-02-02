package streaming

import (
	"context"
	"sync"
)

type ResourceManager struct {
	sem chan struct{}
	mu  sync.Mutex
}

func NewResourceManager(limit int) *ResourceManager {
	return &ResourceManager{
		sem: make(chan struct{}, limit),
	}
}

func (rm *ResourceManager) Acquire(ctx context.Context) error {
	select {
	case rm.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (rm *ResourceManager) Release() {
	select {
	case <-rm.sem:
	default:
		
	}
}

func (rm *ResourceManager) TryAcquire() bool {
	select {
	case rm.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

func (rm *ResourceManager) GetActiveCount() int {
	return len(rm.sem)
}
