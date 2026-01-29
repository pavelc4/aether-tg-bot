package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"

	"github.com/pavelc4/aether-tg-bot/internal/cache"
	"github.com/pavelc4/aether-tg-bot/internal/download"
	"github.com/pavelc4/aether-tg-bot/internal/messaging"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

const (
	MaxAlbumSize = 10
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

func (h *DownloadHandler) Handle(ctx context.Context, e tg.Entities, msg *tg.Message, url string, audioOnly bool) error {
	logger.Info("DownloadHandler Handle called", "url", url, "audioOnly", audioOnly)
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

	cacheKey := fmt.Sprintf("%s|%t", url, audioOnly)
	if cached := cache.GetInstance().Get(cacheKey); cached != nil {
		logger.Info("Cache hit", "url", url)
		var media tg.InputMediaClass
		if cached.Type == cache.TypePhoto {
			media = &tg.InputMediaPhoto{
				ID: &tg.InputPhoto{
					ID:            cached.ID,
					AccessHash:    cached.AccessHash,
					FileReference: cached.FileReference,
				},
			}
		} else {
			media = &tg.InputMediaDocument{
				ID: &tg.InputDocument{
					ID:            cached.ID,
					AccessHash:    cached.AccessHash,
					FileReference: cached.FileReference,
				},
			}
		}

		dummyInfo := provider.VideoInfo{
			Title:    cached.Title,
			FileSize: cached.Size,
		}
		userName := messaging.GetUserName(e, msg)
		
		msgSender := messaging.NewSender(api)
		_, err := msgSender.SendSingle(ctx, inputPeer, &tg.InputReplyToMessage{ReplyToMsgID: msg.ID}, media, dummyInfo, cached.Provider, time.Time{}, url, userName)
		
		if err == nil {
			stats.TrackDownload()
			return nil
		}
		logger.Warn("Failed to send cached media, falling back to download", "error", err)
	}

	infos, providerName, err := provider.Resolve(ctx, url, provider.Options{AudioOnly: audioOnly})
	if err != nil {
		editMsg(fmt.Sprintf("❌ Failed from %s: %v", providerName, err))
		return err
	}

	editMsg(fmt.Sprintf("🚀 starting download from %s (found %d items)", providerName, len(infos)))

	uploader := telegram.NewUploader(api)
	startTime := time.Now()

	downloader := download.NewDownloader(h.streamMgr, uploader)
	finalAlbum, finalInfos := downloader.Download(ctx, infos, audioOnly)

	if len(finalAlbum) == 0 {
		editMsg("❌ No items were successfully downloaded.")
		return nil
	}

	logger.Info("Starting batch send", "total_items", len(finalAlbum))
	
	msgSender := messaging.NewSender(api)
	userName := messaging.GetUserName(e, msg)

	for i := 0; i < len(finalAlbum); i += MaxAlbumSize {
		end := i + MaxAlbumSize
		if end > len(finalAlbum) {
			end = len(finalAlbum)
		}

		batch := finalAlbum[i:end]
		batchInfos := finalInfos[i:end]

		logger.Info("Sending batch", "start", i, "end", end, "count", len(batch))

		var replyTo tg.InputReplyToClass
		if i == 0 {
			replyTo = &tg.InputReplyToMessage{ReplyToMsgID: msg.ID}
		}

		if len(batch) == 1 {
			// Single item
			updates, err := msgSender.SendSingle(ctx, inputPeer, replyTo, batch[0], batchInfos[0], providerName, startTime, url, userName)
			if err != nil {
				logger.Error("Failed to send single media", "error", err)
				editMsg(fmt.Sprintf("❌ Upload Error (Single): %v", err))
			} else {
				logger.Info(" Successfully sent single media")
				if media := getMediaFromUpdates(updates); media != nil {
					media.Title = batchInfos[0].Title
					media.Size = batchInfos[0].FileSize
					media.Provider = providerName
					
					cache.GetInstance().Set(cacheKey, media)
				}
			}
		} else {
			// Album
			err := msgSender.SendAlbum(ctx, inputPeer, replyTo, batch, batchInfos, providerName, startTime, url, userName, i, len(finalAlbum), i)
			if err != nil {
				logger.Error("Failed to send album batch", "error", err)
			}
		}

		if end < len(finalAlbum) {
			time.Sleep(1 * time.Second)
		}
	}

	if sentMsgID != 0 {
		if channelPeer, ok := inputPeer.(*tg.InputPeerChannel); ok {
			logger.Info("Deleting message in channel", "msg_id", sentMsgID)
			_, err := api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
				Channel: &tg.InputChannel{
					ChannelID:  channelPeer.ChannelID,
					AccessHash: channelPeer.AccessHash,
				},
				ID: []int{sentMsgID},
			})
			if err != nil {
				logger.Error("Failed to delete channel message", "error", err)
			}
		} else {
			logger.Info("Deleting message in chat", "msg_id", sentMsgID)
			_, err := api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
				ID:     []int{sentMsgID},
				Revoke: true,
			})
			if err != nil {
				logger.Error("Failed to delete message", "error", err)
			}
		}
	}

	stats.TrackDownload()
	return nil
}
