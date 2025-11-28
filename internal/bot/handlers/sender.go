package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
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

	var photos []string
	var videos []string
	var audios []string
	var docs []string

	for _, path := range filePaths {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp":
			photos = append(photos, path)
		case ".mp4", ".webm", ".mkv":
			videos = append(videos, path)
		case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
			audios = append(audios, path)
		default:
			docs = append(docs, path)
		}
	}

	// Helper to send a group of files as an album
	sendAlbumGroup := func(files []string, isFirstGroup bool) error {
		chunkSize := 10
		for i := 0; i < len(files); i += chunkSize {
			end := i + chunkSize
			if end > len(files) {
				end = len(files)
			}
			batch := files[i:end]

			// Attach caption only to the very first batch of the very first group
			var batchCaption []styling.StyledTextOption
			if isFirstGroup && i == 0 {
				batchCaption = caption
			}

			if err := sendBatch(ctx, client, peer, batch, batchCaption); err != nil {
				return err
			}
		}
		return nil
	}

	// Helper to send files individually (for videos to avoid MEDIA_EMPTY)
	sendIndividual := func(files []string, isFirstGroup bool) error {
		u := uploader.NewUploader(client.API())
		sender := message.NewSender(client.API())

		for i, path := range files {
			info, err := os.Stat(path)
			if err != nil || info.Size() == 0 {
				continue
			}

			log.Printf("Uploading video %d/%d: %s...", i+1, len(files), filepath.Base(path))
			file, err := u.FromPath(ctx, path)
			if err != nil {
				log.Printf("Failed to upload %s: %v", path, err)
				continue
			}

			var opts []styling.StyledTextOption
			if isFirstGroup && i == 0 {
				opts = caption
			}

			log.Printf("Sending video %s...", filepath.Base(path))
			_, err = sender.To(peer).Video(ctx, file, opts...)
			if err != nil {
				log.Printf("Failed to send video %s: %v", path, err)
				return err
			}
		}
		return nil
	}

	// Send in order: Photos (Album), Videos (Individual), Audios (Album), Docs (Album)
	// Caption goes to the first non-empty group
	firstSent := false

	if len(photos) > 0 {
		if err := sendAlbumGroup(photos, !firstSent); err != nil {
			return err
		}
		firstSent = true
	}
	if len(videos) > 0 {
		// Send videos individually
		if err := sendIndividual(videos, !firstSent); err != nil {
			return err
		}
		firstSent = true
	}
	if len(audios) > 0 {
		if err := sendAlbumGroup(audios, !firstSent); err != nil {
			return err
		}
		firstSent = true
	}
	if len(docs) > 0 {
		if err := sendAlbumGroup(docs, !firstSent); err != nil {
			return err
		}
		firstSent = true
	}

	return nil
}

func sendBatch(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption []styling.StyledTextOption) error {
	sender := message.NewSender(client.API())
	u := uploader.NewUploader(client.API())
	var items []message.MultiMediaOption

	for i, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Failed to stat %s: %v", filePath, err)
			continue
		}
		if info.Size() == 0 {
			log.Printf("Skipping empty file: %s", filePath)
			continue
		}

		log.Printf("Uploading album item %d/%d: %s (Size: %d bytes)...", i+1, len(filePaths), filepath.Base(filePath), info.Size())

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
			// Video: Use UploadedDocument with explicit attributes to avoid MEDIA_EMPTY
			items = append(items, message.UploadedDocument(file, itemCaption...).
				MIME("video/mp4").
				Attributes(&tg.DocumentAttributeVideo{
					SupportsStreaming: true,
				}).
				Filename(fileName),
			)

		case ".jpg", ".jpeg", ".png", ".webp":
			// Photo
			items = append(items, message.UploadedPhoto(file, itemCaption...))

		case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
			// Audio
			items = append(items, message.UploadedDocument(file, itemCaption...).
				MIME("audio/mpeg").
				Attributes(&tg.DocumentAttributeAudio{
					Voice: false,
				}).
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

	log.Printf("Sending album with %d items...", len(items))
	_, err := sender.To(peer).Album(ctx, items[0], items[1:]...)
	return err
}
