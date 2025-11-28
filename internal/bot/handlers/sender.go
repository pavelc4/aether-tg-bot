package handlers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"

	"github.com/gotd/td/telegram"
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

	isVideo := false
	if filepath.Ext(filePath) == ".mp4" || filepath.Ext(filePath) == ".webm" {
		isVideo = true
	}

	var media tg.InputMediaClass
	if isVideo {
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
	} else {
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
	}

	_, err = client.API().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  caption,
		RandomID: rand.Int63(),
	})

	return err
}
