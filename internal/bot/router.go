package bot

import (
	"context"
	"strings"

	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/handler"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
)

type Router struct {
	download *handler.DownloadHandler
	admin    *handler.AdminHandler
}

func NewRouter(dl *handler.DownloadHandler, adm *handler.AdminHandler) *Router {
	return &Router{
		download: dl,
		admin:    adm,
	}
}

// OnMessage is the main entry point for updates
func (r *Router) OnMessage(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
	msg, ok := update.Message.(*tg.Message)
	if !ok {
		return nil
	}
	
	text := msg.Message

	if strings.HasPrefix(text, "/stats") {
		return r.admin.HandleStats(ctx, e, msg)
	}
	if strings.HasPrefix(text, "/start") {
		return nil
	}

	url := provider.ExtractURL(text)
	if url != "" {
		if provider.IsSupported(url) {
			return r.download.Handle(ctx, e, msg, url)
		}
	}

	return nil
}
