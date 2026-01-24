package streaming

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/pavelc4/aether-tg-bot/pkg/buffer"
	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
)

type Pipeline struct {
	config Config
	upload func(ctx context.Context, chunk Chunk, fileID int64) error
	update func(read int64, total int64)
}

func NewPipeline(cfg Config, uploadFn func(context.Context, Chunk, int64) error, progressFn func(int64, int64)) *Pipeline {
	return &Pipeline{
		config: cfg,
		upload: uploadFn,
		update: progressFn,
	}
}

func (p *Pipeline) Start(ctx context.Context, input StreamInput, state *StreamState) error {
	body, size, _, err := pkghttp.StreamRequest(ctx, input.URL, input.Headers)
	if err != nil {
		return fmt.Errorf("stream open failed: %w", err)
	}
	defer body.Close()

	if size > 0 {
		state.mu.Lock()
		state.TotalSize = size
		state.TotalParts = int(size/p.config.ChunkSize) + 1
		state.mu.Unlock()
	}
	chunkChan := make(chan Chunk, p.config.BufferSize)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < p.config.UploadWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for chunk := range chunkChan {
				if ctx.Err() != nil {
					return
				}

				var err error
				for attempt := 0; attempt <= p.config.RetryLimit; attempt++ {
					err = p.upload(ctx, chunk, state.FileID)
					if err == nil {
						break
					}

					if ctx.Err() != nil {
						return
					}
				}

				if err != nil {
					select {
					case errChan <- fmt.Errorf("worker %d failed to upload part %d: %w", id, chunk.PartNum, err):
						cancel() // Stop other workers
					default:
					}
					return
				}

				state.mu.Lock()
				state.UploadedParts[chunk.PartNum] = true
				state.mu.Unlock()
				
				if p.update != nil {
					p.update(int64(chunk.Size), state.TotalSize)
				}
				buffer.Put(chunk.Data)
			}
		}(i)
	}

	go func() {
		defer close(chunkChan)
		
		partNum := 0
		for {
			if ctx.Err() != nil {
				return
			}

			buf := buffer.Get()
			if int64(len(buf)) != p.config.ChunkSize {
				// Fallback if pool size doesn't match config
				buf = make([]byte, p.config.ChunkSize)
			}
			
			n, readErr := io.ReadFull(body, buf)
			if n > 0 {
				select {
				case chunkChan <- Chunk{PartNum: partNum, TotalParts: state.TotalParts, Data: buf[:n], Size: n}:
					partNum++
				case <-ctx.Done():
					return
				}
			}

			if readErr != nil {
				if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
					break
				}
				select {
				case errChan <- fmt.Errorf("read failed: %w", readErr):
					cancel()
				default:
				}
				return
			}
		}
	}()

	wg.Wait()
	select {
	case err := <-errChan:
		return err
	default:
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	
	return nil
}
