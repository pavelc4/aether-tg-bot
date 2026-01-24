package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
)

type Uploader struct {
	api *tg.Client
}

func NewUploader(api *tg.Client) *Uploader {
	return &Uploader{api: api}
}

func (u *Uploader) UploadChunk(ctx context.Context, chunk streaming.Chunk, fileID int64, isBig bool) error {	
	totalParts := chunk.TotalParts
	if totalParts <= 0 {
		totalParts = 0 
	}

	if isBig {
		_, err := u.api.UploadSaveBigFilePart(ctx, &tg.UploadSaveBigFilePartRequest{
			FileID:         fileID,
			FilePart:       chunk.PartNum,
			FileTotalParts: totalParts,
			Bytes:          chunk.Data,
		})
		if err != nil {
			return fmt.Errorf("upload big part %d failed: %w", chunk.PartNum, err)
		}
	} else {
		_, err := u.api.UploadSaveFilePart(ctx, &tg.UploadSaveFilePartRequest{
			FileID:   fileID,
			FilePart: chunk.PartNum,
			Bytes:    chunk.Data,
		})
		if err != nil {
			return fmt.Errorf("upload small part %d failed: %w", chunk.PartNum, err)
		}
	}
	
	return nil
}
