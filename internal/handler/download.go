package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type DownloadHandler struct {
	streamMgr *streaming.Manager
	client    *telegram.Client
}

func NewDownloadHandler(sm *streaming.Manager, cli *telegram.Client) *DownloadHandler {
	return &DownloadHandler{
		streamMgr: sm,
		client:    cli,
	}
}

func (h *DownloadHandler) Handle(ctx context.Context, e tg.Entities, msg *tg.Message, url string) error {
	api := h.client.API()
	
	inputPeer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return fmt.Errorf("failed to resolve peer: %w", err)
	}

	sender := message.NewSender(api)
	b := sender.To(inputPeer).Reply(msg.ID)

	sentUpdates, err := b.Text(ctx, "🔎 Detecting...")
	if err != nil {
		return fmt.Errorf("send message failed: %w", err)
	}
	
	sentMsgID := getMsgID(sentUpdates)

	editMsg := func(text string) {
		_, err := sender.To(inputPeer).Edit(sentMsgID).Text(ctx, text)
		if err != nil {
			logger.Error("Failed to edit message", "msg_id", sentMsgID, "error", err)
		}
	}

	info, providerName, err := provider.Resolve(ctx, url)
	if err != nil {
		editMsg(fmt.Sprintf("❌ Failed from %s: %v", providerName, err))
		return err
	}

	editMsg(fmt.Sprintf("🚀 starting download from %s\nFile: %s", providerName, info.FileName))

	input := streaming.StreamInput{
		URL:      info.URL,
		Filename: info.FileName,
		Size:     info.FileSize,
		Headers:  info.Headers,
		MIME:     info.MimeType,
	}

	uploader := telegram.NewUploader(api)

	fileID := time.Now().UnixNano()
	
	uploadFn := func(ctx context.Context, chunk streaming.Chunk, _ int64) error {
		return uploader.UploadChunk(ctx, chunk, fileID)
	}
	
	progressFn := func(uploaded, total int64) {
	}

	err = h.streamMgr.Stream(ctx, input, uploadFn, progressFn)
	
	if err != nil {
		editMsg(fmt.Sprintf("❌ Download failed: %v", err))
		return err
	}


	return h.sendMedia(ctx, sender, inputPeer, input, fileID, sentMsgID)
}

func (h *DownloadHandler) sendMedia(ctx context.Context, sender *message.Sender, peer tg.InputPeerClass, input streaming.StreamInput, fileID int64, replyMsgID int) error {

	inputFile := &tg.InputFileBig{
		ID:    fileID,
		Parts: int(input.Size/config.DefaultChunkSize),
		Name:  input.Filename,
	}
	if input.Size%config.DefaultChunkSize != 0 {
		inputFile.Parts++
	}

	mime := input.MIME
	if mime == "" {
		mime = "video/mp4" // Default
	}

	_, err := sender.To(peer).Reply(replyMsgID).Media(ctx, message.UploadedDocument(inputFile).
		MIME(mime).
		Filename(input.Filename).
		Attributes(&tg.DocumentAttributeVideo{
			SupportsStreaming: true,
		}),
	)

	return err
}

func getMsgID(updates tg.UpdatesClass) int {
	switch u := updates.(type) {
	case *tg.UpdateShortSentMessage:
		return u.ID
	case *tg.Updates:
		for _, update := range u.Updates {
			if msg, ok := update.(*tg.UpdateNewMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID
				}
			}
			if msg, ok := update.(*tg.UpdateNewChannelMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID
				}
			}
		}
	}
	return 0
}
