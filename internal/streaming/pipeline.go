package streaming

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"sync"

	"github.com/pavelc4/aether-tg-bot/pkg/buffer"
	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
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

func (p *Pipeline) Start(ctx context.Context, input StreamInput, state *StreamState) (int, string, error) {
	var body io.ReadCloser
	var size int64
	var err error

	if input.Reader != nil {
		body = input.Reader
		size = input.Size
	} else {
		body, size, _, err = pkghttp.StreamRequest(ctx, input.URL, input.Headers)
		if err != nil {
			return 0, "", fmt.Errorf("stream open failed: %w", err)
		}
	}
	defer body.Close()

	if size <= 0 && input.Size > 0 {
		size = input.Size
	}

	if size > 0 {
		state.mu.Lock()
		state.TotalSize = size
		state.TotalParts = int(size / p.config.ChunkSize)
		if size%p.config.ChunkSize != 0 {
			state.TotalParts++
		}
		state.mu.Unlock()
	} else {
		logger.Error("Stream size unknown, MTProto upload would fail", "url", input.URL)
		return 0, "", fmt.Errorf("unknown stream size (required for MTProto)")
	}
	
	chunkChan := make(chan Chunk, p.config.BufferSize)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	numWorkers := 1
	if state.TotalSize > 0 {
		calculated := int(state.TotalSize / (1 * 1024 * 1024))
		if calculated > p.config.MaxUploadWorkers {
			numWorkers = p.config.MaxUploadWorkers
		} else if calculated < p.config.MinUploadWorkers {
			numWorkers = p.config.MinUploadWorkers
		} else {
			numWorkers = calculated
		}
	} else {
		numWorkers = p.config.MinUploadWorkers
	}
	if numWorkers < 1 { 
		numWorkers = 1 
	}

	logger.Info("Starting upload pipeline", 
		"workers", numWorkers, 
		"size_mb", state.TotalSize/1024/1024,
	)

	for i := 0; i < numWorkers; i++ {
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

	totalParts := 0
	md5Result := ""
	
	go func() {
		defer close(chunkChan)
		
		hasher := md5.New()
		partNum := 0
		for {
			if ctx.Err() != nil {
				return
			}

			buf := buffer.Get()
			if int64(len(buf)) != p.config.ChunkSize {
				buf = make([]byte, p.config.ChunkSize)
			}
			
			n, readErr := io.ReadFull(body, buf)
			if n > 0 {
				// Write to haser
				hasher.Write(buf[:n])
				
				select {
				case chunkChan <- Chunk{PartNum: partNum, TotalParts: state.TotalParts, Data: buf[:n], Size: n}:
					partNum++
					totalParts = partNum
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
		md5Result = fmt.Sprintf("%x", hasher.Sum(nil))
	}()

	wg.Wait()
	select {
	case err := <-errChan:
		return totalParts, "", err
	default:
		if ctx.Err() != nil {
			return totalParts, "", ctx.Err()
		}
	}
	
	if totalParts == 0 {
		return 0, "", fmt.Errorf("stream returned no data")
	}

	return totalParts, md5Result, nil
}
