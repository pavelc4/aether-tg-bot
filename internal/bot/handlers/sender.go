package handlers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"strings"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

func SendMedia(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePath string, caption string) error {
	u := uploader.NewUploader(client.API())

	log.Printf("Uploading %s...", filepath.Base(filePath))

	file, err := u.FromPath(ctx, filePath)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	fileName := filepath.Base(filePath)

	ext := strings.ToLower(filepath.Ext(filePath))
	var media tg.InputMediaClass

	switch ext {
	case ".mp4", ".webm", ".mkv":
		media = &tg.InputMediaUploadedDocument{
			File:     file,
			MimeType: "video/mp4",
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: fileName},
				&tg.DocumentAttributeVideo{
					SupportsStreaming: true,
				},
			},
		}
	case ".jpg", ".jpeg", ".png", ".webp":
		media = &tg.InputMediaUploadedPhoto{
			File: file,
		}
	case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
		media = &tg.InputMediaUploadedDocument{
			File:     file,
			MimeType: "audio/mpeg",
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: fileName},
				&tg.DocumentAttributeAudio{
					Voice: false,
				},
			},
		}
	default:
		media = &tg.InputMediaUploadedDocument{
			File:     file,
			MimeType: "application/octet-stream",
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeFilename{FileName: fileName},
			},
		}
	}

	_, err = client.API().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  caption,
		RandomID: rand.Int63(),
	})

	return err
}

func SendFiles(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption string) error {
	var visuals, audios, docs []string

	for _, fp := range filePaths {
		ext := strings.ToLower(filepath.Ext(fp))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp", ".mp4", ".webm", ".mkv":
			visuals = append(visuals, fp)
		case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
			audios = append(audios, fp)
		default:
			docs = append(docs, fp)
		}
	}

	// Helper to process a group of files
	processGroup := func(files []string) error {
		// Chunk into groups of 10
		chunkSize := 10
		for i := 0; i < len(files); i += chunkSize {
			end := i + chunkSize
			if end > len(files) {
				end = len(files)
			}
			batch := files[i:end]

			if len(batch) == 1 {
				// Send as single media
				err := SendMedia(ctx, client, peer, batch[0], caption)
				if err != nil {
					log.Printf("Failed to send single media %s: %v", batch[0], err)
				}
			} else {
				// Send as album using message.Sender
				err := sendBatch(ctx, client, peer, batch, caption)
				if err != nil {
					log.Printf("Failed to send batch: %v", err)
				}
			}
		}
		return nil
	}

	// Process each type
	// Visuals (Photos/Videos) can be mixed in an album
	if len(visuals) > 0 {
		processGroup(visuals)
	}
	// Audios can be grouped (playlist)
	if len(audios) > 0 {
		processGroup(audios)
	}
	// Docs can be grouped
	if len(docs) > 0 {
		processGroup(docs)
	}

	return nil
}

func sendBatch(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption string) error {
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
		if i == 0 && caption != "" {
			itemCaption = append(itemCaption, styling.Plain(caption))
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
