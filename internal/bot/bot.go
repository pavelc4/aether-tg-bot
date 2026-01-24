package bot

import (
	"context"

	"github.com/pavelc4/aether-tg-bot/internal/telegram"
)

type Bot struct {
	client *telegram.Client
	router *Router
}

func New(client *telegram.Client, router *Router) *Bot {
	return &Bot{
		client: client,
		router: router,
	}
}

func (b *Bot) Run(ctx context.Context, token string) error {
	return b.client.Start(ctx, token)
}
