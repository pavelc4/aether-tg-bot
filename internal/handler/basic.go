package handler

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"
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

	helpText := fmt.Sprintf(
		"<b>Aether Downloader Bot</b>\n\n" +
			"I can help you download media from various platforms.\n\n" +
			"<b>Available Commands</b>\n" +
			"├ <code>/dl [URL]</code> - Download content\n" +
			"├ <code>/mp [URL]</code> - Download audio only\n" +
			"├ <code>/video [URL]</code> - Download video only\n" +
			"├ <code>/speedtest</code> - Check server speed\n" +
			"└ <code>/help</code> - Show this help message\n\n" +
			"<b>Quick Tips</b>\n" +
			"• Just send a URL to download video automatically\n" +
			"• Supports <b>YouTube, TikTok, Instagram, X</b>, and more!\n" +
			"• Fast multithreaded downloads\n\n" +
			"<i>Fun fact: This bot is written in Go</i> 🐹",
	)

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

	_, err = sender.To(peer).Markup(&markup).StyledText(ctx, html.String(nil, helpText))
	return err
}

func (h *BasicHandler) HandleUnknown(ctx context.Context, e tg.Entities, msg *tg.Message) error {
	peer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}

	sender := message.NewSender(h.client.API())
	_, err = sender.To(peer).Reply(msg.ID).Text(ctx, "Unknown command. Try /help to see available commands.")
	return err
}
