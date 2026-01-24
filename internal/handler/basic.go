package handler

import (
	"context"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
)

type BasicHandler struct {
	client *telegram.Client
}

func NewBasicHandler(cli *telegram.Client) *BasicHandler {
	return &BasicHandler{client: cli}
}

func (h *BasicHandler) HandleStart(ctx context.Context, e tg.Entities, msg *tg.Message) error {
	peer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}
	
	sender := message.NewSender(h.client.API())
	_, err = sender.To(peer).Text(ctx, "👋 Welcome to Aether Bot (Gotd Edition)!\n\nSend me a link from TikTok, Instagram, YouTube, etc. to download.")
	return err
}

func (h *BasicHandler) HandleHelp(ctx context.Context, e tg.Entities, msg *tg.Message) error {
	peer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}

	helpText := `
**Aether Downloader Bot**

I can help you download media from various platforms.

**Available Commands:**
• /dl [URL] - Download content
• /mp [URL] - Download audio only
• /video [URL] - Download video only
• /start - Start the bot
• /help - Show this help message
• /stats - Show bot statistics (owner only)

**Quick Tips:**
• Just send a URL to download video
• Bot uses Cobalt API first, then falls back to yt-dlp
• Multithreaded downloads with 16 concurrent threads
• Real-time progress tracking

**Supported Platforms:**
YouTube, TikTok, Instagram, X, and more!

Fun fact: This bot is written in Go 🐹
`
	sender := message.NewSender(h.client.API())
	markup := tg.ReplyInlineMarkup{
		Rows: []tg.KeyboardButtonRow{
			{
				Buttons: []tg.KeyboardButtonClass{
					&tg.KeyboardButtonURL{
						Text: "Developer",
						URL:  "https://t.me/pavellc",
					},
					&tg.KeyboardButtonURL{
						Text: "Source",
						URL:  "https://github.com/pavelc4/aether-tg-bot",
					},
				},
			},
		},
	}

	_, err = sender.To(peer).Markup(&markup).Text(ctx, helpText)
	return err
}
