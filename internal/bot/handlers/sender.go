package handlers

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

// SendFiles sends multiple files, grouping them by type and chunking them into albums.
// It supports styled captions.
func SendFiles(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption []styling.StyledTextOption) error {
	if len(filePaths) == 0 {
		return nil
	}

	// Group files by type
	var visuals []string // Photos and Videos
	var audios []string
	var docs []string

	for _, path := range filePaths {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".mp4", ".webm", ".mkv":
			visuals = append(visuals, path)
		case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
			audios = append(audios, path)
		default:
			docs = append(docs, path)
		}
	}

	// Send Visuals (Album)
	if len(visuals) > 0 {
		if err := sendBatchedFiles(ctx, client, peer, visuals, caption); err != nil {
			return err
		}
	}

	// Send Audios
	if len(audios) > 0 {
		// For audio, we might want to send caption with the first one too?
		// Yes, sendBatchedFiles handles that.
		if err := sendBatchedFiles(ctx, client, peer, audios, caption); err != nil {
			return err
		}
	}

	// Send Docs
	if len(docs) > 0 {
		if err := sendBatchedFiles(ctx, client, peer, docs, caption); err != nil {
			return err
		}
	}

	return nil
}

func sendBatchedFiles(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption []styling.StyledTextOption) error {
	batchSize := 10
	for i := 0; i < len(filePaths); i += batchSize {
		end := i + batchSize
		if end > len(filePaths) {
			end = len(filePaths)
		}

		batch := filePaths[i:end]
		// Only send caption with the first batch
		var batchCaption []styling.StyledTextOption
		if i == 0 {
			batchCaption = caption
		}

		if err := sendBatch(ctx, client, peer, batch, batchCaption); err != nil {
			return err
		}
	}
	return nil
}

func sendBatch(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption []styling.StyledTextOption) error {
	sender := message.NewSender(client.API())
	u := uploader.NewUploader(client.API())
	var items []message.MultiMediaOption

	for i, filePath := range filePaths {
		log.Printf("Uploading album item %d/%d: %s...", i+1, len(filePaths), filepath.Base(filePath))

		file, err := u.FromPath(ctx, filePath)
		if err != nil {
			log.Printf("Failed to upload %s: %v", filePath, err)
			continue
		}

		fileName := filepath.Base(filePath)
		ext := strings.ToLower(filepath.Ext(filePath))

		// Only add caption to the first item
		var itemCaption []styling.StyledTextOption
		if i == 0 && len(caption) > 0 {
			itemCaption = caption
		}

		switch ext {
		case ".mp4", ".webm", ".mkv":
			// Video with streaming support
			items = append(items, message.Video(file, itemCaption...).
				SupportsStreaming(),
			)

		case ".jpg", ".jpeg", ".png", ".webp":
			// Photo
			items = append(items, message.UploadedPhoto(file, itemCaption...))

		case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
			// Audio
			items = append(items, message.Audio(file, itemCaption...).
				Filename(fileName),
			)

		default:
			// Document
			items = append(items, message.UploadedDocument(file, itemCaption...).
				Filename(fileName),
			)
		}
	}

	if len(items) == 0 {
		return fmt.Errorf("no media to send")
	}
	_, err := sender.To(peer).Album(ctx, items[0], items[1:]...)
	return err
}
