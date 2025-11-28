package handlers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/downloader"
	"github.com/pavelc4/aether-tg-bot/internal/downloader/core"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
)

type Handler struct {
	Client *telegram.Client
	Sender *message.Sender
}

func NewHandler(client *telegram.Client) *Handler {
	return &Handler{
		Client: client,
		Sender: message.NewSender(client.API()),
	}
}

func (h *Handler) Register(dispatcher tg.UpdateDispatcher) {
	dispatcher.OnNewMessage(func(ctx context.Context, entities tg.Entities, update *tg.UpdateNewMessage) error {
		msg, ok := update.Message.(*tg.Message)
		if !ok {
			return nil
		}

		// Handle commands
		if strings.HasPrefix(msg.Message, "/start") {
			return h.handleStart(ctx, msg, entities)
		}
		if strings.HasPrefix(msg.Message, "/help") {
			return h.handleHelp(ctx, msg, entities)
		}
		if strings.HasPrefix(msg.Message, "/stats") {
			return h.handleStats(ctx, msg, entities)
		}

		// Handle URL (Download)
		if strings.HasPrefix(msg.Message, "http") {
			return h.handleDownload(ctx, msg, entities, msg.Message, false)
		}

		// Handle explicit download commands
		if strings.HasPrefix(msg.Message, "/dl") {
			return h.handleDL(ctx, msg, entities)
		}
		if strings.HasPrefix(msg.Message, "/mp") {
			return h.handleMP(ctx, msg, entities)
		}
		if strings.HasPrefix(msg.Message, "/video") {
			return h.handleVideo(ctx, msg, entities)
		}

		return nil
	})
}

func resolvePeer(entities tg.Entities, peer tg.PeerClass) (tg.InputPeerClass, error) {
	switch p := peer.(type) {
	case *tg.PeerUser:
		user, ok := entities.Users[p.UserID]
		if !ok {
			return nil, fmt.Errorf("user not found in entities")
		}
		return user.AsInputPeer(), nil
	case *tg.PeerChat:
		chat, ok := entities.Chats[p.ChatID]
		if !ok {
			return nil, fmt.Errorf("chat not found in entities")
		}
		return chat.AsInputPeer(), nil
	case *tg.PeerChannel:
		channel, ok := entities.Channels[p.ChannelID]
		if !ok {
			return nil, fmt.Errorf("channel not found in entities")
		}
		return channel.AsInputPeer(), nil
	default:
		return nil, fmt.Errorf("unknown peer type")
	}
}

func isValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func (h *Handler) handleStart(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	peer, err := resolvePeer(entities, msg.PeerID)
	if err != nil {
		return err
	}
	_, err = h.Sender.To(peer).Text(ctx, "👋 Welcome to Aether Bot (Gotd Edition)!\n\nSend me a link from TikTok, Instagram, YouTube, etc. to download.")
	return err
}

func (h *Handler) handleHelp(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	peer, err := resolvePeer(entities, msg.PeerID)
	if err != nil {
		return err
	}
	helpText := `
		Aether Downloader Bot

		I can help you download media from various platforms.

		Available Commands:
		• /dl [URL] - Download content
		• /mp [URL] - Download audio only
		• /video [URL] - Download video only
		• /start - Start the bot
		• /help - Show this help message
		• /stats - Show bot statistics (owner only)

		Quick Tips:
		• Just send a URL to download video
		• Bot uses Cobalt API first, then falls back to yt-dlp
		• Multithreaded downloads with 16 concurrent threads
		• Real-time progress tracking

		Supported Platforms:
		YouTube, TikTok, Instagram, X, and more!

		Fun fact: This bot is written in Go 🐹
    `

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

	_, err = h.Sender.To(peer).Markup(&markup).Text(ctx, helpText)
	return err
}

func (h *Handler) handleDL(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	parts := strings.Fields(msg.Message)
	if len(parts) < 2 {
		peer, err := resolvePeer(entities, msg.PeerID)
		if err != nil {
			return err
		}
		_, err = h.Sender.To(peer).Text(ctx, "❌ Usage: /dl <url>\n\nExample: /dl https://youtu.be/example")
		return err
	}

	targetURL := parts[1]
	if !isValidURL(targetURL) {
		peer, err := resolvePeer(entities, msg.PeerID)
		if err != nil {
			return err
		}
		_, err = h.Sender.To(peer).Text(ctx, "❌ Invalid URL format. Please provide a valid URL.")
		return err
	}

	return h.handleDownload(ctx, msg, entities, targetURL, false)
}

func (h *Handler) handleMP(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	parts := strings.Fields(msg.Message)
	if len(parts) < 2 {
		peer, err := resolvePeer(entities, msg.PeerID)
		if err != nil {
			return err
		}
		_, err = h.Sender.To(peer).Text(ctx, "❌ Usage: /mp <url>\n\nExample: /mp https://youtu.be/example")
		return err
	}

	targetURL := parts[1]
	if !isValidURL(targetURL) {
		peer, err := resolvePeer(entities, msg.PeerID)
		if err != nil {
			return err
		}
		_, err = h.Sender.To(peer).Text(ctx, "❌ Invalid URL format. Please provide a valid URL.")
		return err
	}

	return h.handleDownload(ctx, msg, entities, targetURL, true)
}

func (h *Handler) handleVideo(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	parts := strings.Fields(msg.Message)
	if len(parts) < 2 {
		peer, err := resolvePeer(entities, msg.PeerID)
		if err != nil {
			return err
		}
		_, err = h.Sender.To(peer).Text(ctx, "❌ Usage: /video <url>\n\nExample: /video https://youtu.be/example")
		return err
	}

	targetURL := parts[1]
	if !isValidURL(targetURL) {
		peer, err := resolvePeer(entities, msg.PeerID)
		if err != nil {
			return err
		}
		_, err = h.Sender.To(peer).Text(ctx, "❌ Invalid URL format. Please provide a valid URL.")
		return err
	}

	return h.handleDownload(ctx, msg, entities, targetURL, false)
}

func (h *Handler) handleStats(ctx context.Context, msg *tg.Message, entities tg.Entities) error {
	ownerIDStr := os.Getenv("OWNER_ID")
	if ownerIDStr == "" {
		return nil // Silently ignore if no owner configured
	}

	ownerID, err := strconv.ParseInt(ownerIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid OWNER_ID format: %v", err)
		return nil
	}

	// Check authorization
	if peer, ok := msg.PeerID.(*tg.PeerUser); ok {
		if int64(peer.UserID) != ownerID {
			return nil // Silently ignore non-owners
		}
	} else {
		return nil
	}

	peer, err := resolvePeer(entities, msg.PeerID)
	if err != nil {
		return err
	}

	s := stats.GetStats()
	sysInfo, err := stats.GetSystemInfo()
	if err != nil {
		errMsg := fmt.Sprintf("❌ Failed to get system info: %v", err)
		_, textErr := h.Sender.To(peer).Text(ctx, errMsg)
		return textErr
	}

	statsMsg := fmt.Sprintf(
		"🖥️ System Information\n"+
			"├─ OS: `%s`\n"+
			"├─ Hostname: `%s`\n"+
			"└─ Uptime: `%s`\n\n"+
			"⚙️ CPU\n"+
			"├─ Cores: `%d`\n"+
			"└─ Usage: `%.2f%%`\n\n"+
			"💾 Memory\n"+
			"├─ Used: `%s / %s (%.1f%%)`\n"+
			"└─ Available: `%s`\n\n"+
			"💿 Disk (/)\n"+
			"├─ Used: `%s / %s (%.1f%%)`\n"+
			"└─ Free: `%s`\n\n"+
			"🌐 Network\n"+
			"├─ Sent: `%s`\n"+
			"└─ Received: `%s`\n\n"+
			"🐹 Bot Process\n"+
			"├─ Uptime: `%s`\n"+
			"├─ PID: `%d`\n"+
			"├─ CPU: `%.2f%%`\n"+
			"├─ Memory: `%s`\n"+
			"└─ Go Version: `%s`\n\n"+
			"🔧 Go Runtime\n"+
			"├─ Goroutines: `%d`\n"+
			"├─ Heap Alloc: `%s`\n"+
			"└─ GC Runs: `%d`\n\n"+
			"📊 Download Stats\n"+
			"├─ Today: `%s`\n"+
			"├─ This Week: `%s`\n"+
			"└─ This Month: `%s`",
		sysInfo.OS,
		sysInfo.Hostname,
		formatUptime(sysInfo.SystemUptime),
		sysInfo.CPUCores,
		sysInfo.CPUUsage,
		FormatFileSize(int64(sysInfo.MemUsed)), FormatFileSize(int64(sysInfo.MemTotal)), sysInfo.MemPercent,
		FormatFileSize(int64(sysInfo.MemAvailable)),
		FormatFileSize(int64(sysInfo.DiskUsed)), FormatFileSize(int64(sysInfo.DiskTotal)), sysInfo.DiskPercent,
		FormatFileSize(int64(sysInfo.DiskFree)),
		FormatFileSize(int64(sysInfo.NetSent)),
		FormatFileSize(int64(sysInfo.NetRecv)),
		formatUptime(sysInfo.ProcessUptime),
		sysInfo.ProcessPID,
		sysInfo.ProcessCPU,
		FormatFileSize(int64(sysInfo.ProcessMem)),
		sysInfo.GoVersion,
		sysInfo.Goroutines,
		FormatFileSize(int64(sysInfo.HeapAlloc)),
		sysInfo.GCRuns,
		formatPeriodStats(s.GetPeriodStats("today")),
		formatPeriodStats(s.GetPeriodStats("week")),
		formatPeriodStats(s.GetPeriodStats("month")),
	)

	_, err = h.Sender.To(peer).Text(ctx, statsMsg)
	return err
}

func getMsgID(updates tg.UpdatesClass) (int, bool) {
	switch u := updates.(type) {
	case *tg.UpdateShortSentMessage:
		return u.ID, true
	case *tg.Updates:
		for _, update := range u.Updates {
			if msg, ok := update.(*tg.UpdateNewMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID, true
				}
			}
		}
	case *tg.UpdatesCombined:
		for _, update := range u.Updates {
			if msg, ok := update.(*tg.UpdateNewMessage); ok {
				if m, ok := msg.Message.(*tg.Message); ok {
					return m.ID, true
				}
			}
		}
	}
	return 0, false
}

func (h *Handler) handleDownload(ctx context.Context, msg *tg.Message, entities tg.Entities, url string, audioOnly bool) error {
	peer, err := resolvePeer(entities, msg.PeerID)
	if err != nil {
		return err
	}

	// Validate URL
	if !isValidURL(url) {
		_, err = h.Sender.To(peer).Text(ctx, "❌ Invalid URL format. Please provide a valid URL.")
		return err
	}

	// Send "Processing..."
	updates, err := h.Sender.To(peer).Text(ctx, "⏳ Processing...")
	if err != nil {
		return err
	}

	msgID, hasMsgID := getMsgID(updates)

	// Track user
	if userPeer, ok := msg.PeerID.(*tg.PeerUser); ok {
		stats.TrackUser(int64(userPeer.UserID))
	}

	// Download
	startTime := time.Now()
	filePaths, totalSize, providerName, title, err := downloader.UniversalDownload(url, audioOnly, 0)
	duration := time.Since(startTime)

	if err != nil {
		errMsg := fmt.Sprintf("❌ Error: %v", err)
		if hasMsgID {
			_, editErr := h.Client.API().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
				Peer:    peer,
				ID:      msgID,
				Message: errMsg,
			})
			if editErr != nil {
				log.Printf("Failed to edit error message: %v", editErr)
				// Fallback: send new message
				h.Sender.To(peer).Text(ctx, errMsg)
			}
		} else {
			h.Sender.To(peer).Text(ctx, errMsg)
		}
		return err
	}
	defer core.CleanupFiles(filePaths)

	// Update to "Uploading..."
	if hasMsgID {
		_, editErr := h.Client.API().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
			Peer:    peer,
			ID:      msgID,
			Message: "🚀 Uploading...",
		})
		if editErr != nil {
			log.Printf("Failed to edit to uploading message: %v", editErr)
		}
	}

	// Get User Name
	userName := "Unknown"
	if userPeer, ok := msg.PeerID.(*tg.PeerUser); ok {
		// Try to find user in entities
		for _, user := range entities.Users {
			if user.ID == userPeer.UserID {
				if user.Username != "" {
					userName = "@" + user.Username
				} else {
					userName = user.FirstName
					if user.LastName != "" {
						userName += " " + user.LastName
					}
				}
				break
			}
		}
	}

	displayProvider := providerName
	if strings.Contains(url, "instagram.com") {
		displayProvider = "Instagram"
	} else if strings.Contains(url, "youtube.com") {
		displayProvider = "YouTube"
	} else if strings.Contains(url, "tiktok.com") {
		displayProvider = "TikTok"
	} else if strings.Contains(url, "twitter.com") || strings.Contains(url, "x.com") {
		displayProvider = "X (Twitter)"
	} else if strings.Contains(url, "facebook.com") || strings.Contains(url, "fb.watch") {
		displayProvider = "Facebook"
	}

	captionOpts := []styling.StyledTextOption{
		styling.Bold(title), styling.Plain("\n"),
		styling.Plain("🔗 Source : "), styling.TextURL(displayProvider, url), styling.Plain("\n"),
		styling.Plain("💾 Size : "), styling.Plain(FormatFileSize(totalSize)), styling.Plain("\n"),
		styling.Plain("⏱️ Processing Time : "), styling.Plain(duration.Round(time.Second).String()), styling.Plain("\n"),
		styling.Plain("👤 By : "), styling.Plain(userName),
	}

	err = SendFiles(ctx, h.Client, peer, filePaths, captionOpts)

	if hasMsgID {
		_, delErr := h.Client.API().MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
			ID:     []int{msgID},
			Revoke: true,
		})
		if delErr != nil {
			log.Printf("Failed to delete uploading message: %v", delErr)
			if err == nil {
				h.Client.API().MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
					Peer:    peer,
					ID:      msgID,
					Message: "✅ Upload complete!",
				})
			}
		}
	}

	if err != nil {
		log.Printf("Upload failed: %v", err)
		errMsg := fmt.Sprintf("❌ Upload failed: %v", err)
		h.Sender.To(peer).Text(ctx, errMsg)
		return err
	}

	stats.TrackDownload()
	return nil
}
