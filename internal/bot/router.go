package bot

import (
	"context"
	"strings"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/handler"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type Router struct {
	download *handler.DownloadHandler
	admin    *handler.AdminHandler
	basic    *handler.BasicHandler
}

func NewRouter(dl *handler.DownloadHandler, adm *handler.AdminHandler, basic *handler.BasicHandler) *Router {
	return &Router{
		download: dl,
		admin:    adm,
		basic:    basic,
	}
}

// OnMessage is the main entry point for updates
func (r *Router) OnMessage(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
	msg, ok := update.Message.(*tg.Message)
	if !ok {
		return nil
	}
	
	text := msg.Message

	if strings.HasPrefix(text, "/start") {
		return r.basic.HandleStart(ctx, e, msg)
	}
	if strings.HasPrefix(text, "/help") {
		return r.basic.HandleHelp(ctx, e, msg)
	}
	if strings.HasPrefix(text, "/stats") {
		return r.admin.HandleStats(ctx, e, msg)
	}
	if strings.HasPrefix(text, "/speedtest") || strings.HasPrefix(text, "/speed") {
		return r.basic.HandleSpeedtest(ctx, e, msg)
	}

	if strings.HasPrefix(text, "/dl") || strings.HasPrefix(text, "/video") {
		parts := strings.Fields(text)
		if len(parts) > 1 {
			url := parts[1]
			if provider.ExtractURL(url) != "" && provider.IsSupported(url) {
				return r.download.Handle(ctx, e, msg, url, false)
			}
		}
	}
	if strings.HasPrefix(text, "/mp") {
		parts := strings.Fields(text)
		if len(parts) > 1 {
			url := parts[1]
			extracted := provider.ExtractURL(url)
			supported := provider.IsSupported(url)
			logger.Info("Checking /mp command", "url", url, "extracted", extracted, "supported", supported)
			
			if extracted != "" && supported {
				return r.download.Handle(ctx, e, msg, url, true)
			}
		}
	}

	url := provider.ExtractURL(text)
	if url != "" {
		if provider.IsSupported(url) {
			return r.download.Handle(ctx, e, msg, url, false)
		}
	}

	return nil
}
