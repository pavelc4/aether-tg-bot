package handler

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

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

func (h *BasicHandler) HandleSpeedtest(ctx context.Context, e tg.Entities, msg *tg.Message) error {
	peer, err := resolvePeer(msg.PeerID, e)
	if err != nil {
		return err
	}
	
	sender := message.NewSender(h.client.API())
	b := sender.To(peer).Reply(msg.ID)
	
	u, err := b.Text(ctx, " Testing Latency...")
	if err != nil {
		return err
	}
	
	msgID := 0
	if up, ok := u.(*tg.UpdateShortSentMessage); ok {
		msgID = up.ID
	} else if ups, ok := u.(*tg.Updates); ok {
		for _, update := range ups.Updates {
			if m, ok := update.(*tg.UpdateNewMessage); ok {
				if msg, ok := m.Message.(*tg.Message); ok {
					msgID = msg.ID
				}
			}
		}
	}

	var pings []float64
	client := &http.Client{Timeout: 5 * time.Second}
	
	for i := 0; i < 5; i++ {
		pStart := time.Now()
		_, err := client.Head("https://speed.cloudflare.com/__down?bytes=0")
		if err == nil {
			pings = append(pings, float64(time.Since(pStart).Milliseconds()))
		}
		time.Sleep(100 * time.Millisecond)
	}

	var avgPing, jitter float64
	if len(pings) > 0 {
		var sum float64
		for _, p := range pings {
			sum += p
		}
		avgPing = sum / float64(len(pings))
		
		var sumDiffSq float64
		for _, p := range pings {
			diff := p - avgPing
			sumDiffSq += diff * diff
		}
		jitter = math.Sqrt(sumDiffSq / float64(len(pings)))
	}

	sender.To(peer).Edit(msgID).Text(ctx, "Testing Download (Cloudflare)...")

	start := time.Now()
	resp, err := http.Get("https://speed.cloudflare.com/__down?bytes=52428800")
	if err != nil {
		sender.To(peer).Edit(msgID).Text(ctx, fmt.Sprintf("Speedtest Failed: %v", err))
		return err
	}
	defer resp.Body.Close()
	
	written, _ := io.Copy(io.Discard, resp.Body)
	duration := time.Since(start)
	
	mbps := (float64(written) * 8) / (duration.Seconds() * 1000 * 1000)
	
	result := fmt.Sprintf(
		"<b>Results Testing </b>\n\n"+
		"<b>Ping:</b> <code>%.2f ms</code>\n"+
		"<b>Jitter:</b> <code>%.2f ms</code>\n"+
		"<b>Download:</b> <code>%.2f Mbps</code>\n"+
		"<b>Time:</b> <code>%.2fs</code>\n"+
		"<b>Size:</b> <code>%.2f MB</code>",
		avgPing,
		jitter,
		mbps,
		duration.Seconds(),
		float64(written)/1024/1024,
	)
	
	_, err = sender.To(peer).Edit(msgID).StyledText(ctx, html.String(nil, result))
	return err
}
