package streaming

import (
	"sync"
)

type StateManager struct {
	states sync.Map
}

func NewStateManager() *StateManager {
	return &StateManager{}
}

func (sm *StateManager) NewState(streamID string, fileID int64, totalSize int64) *StreamState {
	state := &StreamState{
		FileID:        fileID,
		TotalSize:     totalSize,
		UploadedParts: make(map[int]bool),
		ChunkRetries:  make(map[int]int),
	}
	sm.states.Store(streamID, state)
	return state
}

func (sm *StateManager) GetState(streamID string) (*StreamState, bool) {
	val, ok := sm.states.Load(streamID)
	if !ok {
		return nil, false
	}
	return val.(*StreamState), true
}

func (sm *StateManager) DeleteState(streamID string) {
	sm.states.Delete(streamID)
}

func (sm *StateManager) MarkPartUploaded(streamID string, partNum int) {
	if state, ok := sm.GetState(streamID); ok {
		state.mu.Lock()
		defer state.mu.Unlock()
		state.UploadedParts[partNum] = true
	}
}
