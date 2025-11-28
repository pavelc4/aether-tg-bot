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

func SendFiles(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption []styling.StyledTextOption) error {
	if len(filePaths) == 0 {
		return nil
	}

	groups := map[string][]string{
		"photos": {},
		"videos": {},
		"audios": {},
		"docs":   {},
	}

	for _, path := range filePaths {
		ext := strings.ToLower(filepath.Ext(path))
		grp, ok := extGroups[ext]
		if !ok {
			grp = "docs"
		}
		groups[grp] = append(groups[grp], path)
	}

	order := []fileGroup{
		{"photos", groups["photos"]},
		{"videos", groups["videos"]},
		{"audios", groups["audios"]},
		{"docs", groups["docs"]},
	}

	first := true

	for _, g := range order {
		if len(g.files) == 0 {
			continue
		}

		var err error

		attachCaption := first
		if g.name == "videos" {
			err = sendIndividualVideos(ctx, client, peer, g.files, caption, attachCaption)
		} else {
			err = sendAlbumGroup(ctx, client, peer, g.files, caption, attachCaption)
		}

		if err != nil {
			return err
		}
		first = false
	}

	return nil
}

func sendAlbumGroup(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, files []string, caption []styling.StyledTextOption, withCaption bool) error {
	chunk := 10
	for i := 0; i < len(files); i += chunk {
		end := i + chunk
		if end > len(files) {
			end = len(files)
		}

		batch := files[i:end]
		var capUse []styling.StyledTextOption

		if withCaption && i == 0 {
			capUse = caption
		}

		if err := sendBatch(ctx, client, peer, batch, capUse); err != nil {
			return err
		}
	}

	return nil
}

func sendIndividualVideos(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, files []string, caption []styling.StyledTextOption, withCaption bool) error {
	u := uploader.NewUploader(client.API())
	sender := message.NewSender(client.API())

	for i, path := range files {
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			continue
		}

		file, err := u.FromPath(ctx, path)
		if err != nil {
			log.Printf("Upload failed: %s", err)
			continue
		}

		var opts []styling.StyledTextOption
		if withCaption && i == 0 {
			opts = caption
		}

		_, err = sender.To(peer).Video(ctx, file, opts...)
		if err != nil {
			return err
		}
	}
	return nil
}

func sendBatch(ctx context.Context, client *telegram.Client, peer tg.InputPeerClass, filePaths []string, caption []styling.StyledTextOption) error {
	u := uploader.NewUploader(client.API())
	sender := message.NewSender(client.API())

	var items []message.MultiMediaOption

	for i, p := range filePaths {
		info, err := os.Stat(p)
		if err != nil || info.Size() == 0 {
			continue
		}

		file, err := u.FromPath(ctx, p)
		if err != nil {
			continue
		}

		name := filepath.Base(p)
		ext := strings.ToLower(filepath.Ext(p))

		var capUse []styling.StyledTextOption
		if i == 0 {
			capUse = caption
		}

		switch ext {
		case ".mp4", ".webm", ".mkv":
			items = append(items,
				message.UploadedDocument(file, capUse...).
					MIME("video/mp4").
					Attributes(&tg.DocumentAttributeVideo{SupportsStreaming: true}).
					Filename(name),
			)
		case ".jpg", ".jpeg", ".png", ".webp":
			items = append(items, message.UploadedPhoto(file, capUse...))
		case ".mp3", ".m4a", ".ogg", ".flac", ".wav", ".opus":
			items = append(items,
				message.UploadedDocument(file, capUse...).
					MIME("audio/mpeg").
					Attributes(&tg.DocumentAttributeAudio{}).
					Filename(name),
			)
		default:
			items = append(items, message.UploadedDocument(file, capUse...).Filename(name))
		}
	}

	if len(items) == 0 {
		return fmt.Errorf("no media to send")
	}

	_, err := sender.To(peer).Album(ctx, items[0], items[1:]...)
	return err
}
