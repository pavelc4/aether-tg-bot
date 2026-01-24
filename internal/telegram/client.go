package telegram

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type Client struct {
	client     *telegram.Client
	api        *tg.Client
	dispatcher tg.UpdateDispatcher
	me         *tg.User
}

func NewClient(cfg *config.Config, dispatcher tg.UpdateDispatcher) (*Client, error) {
	sessionPath := filepath.Join(cfg.SessionDir, "session.json")
	
	opts := telegram.Options{
		SessionStorage: &session.FileStorage{Path: sessionPath},
		UpdateHandler:  dispatcher,
	}

	client := telegram.NewClient(cfg.AppID, cfg.AppHash, opts)
	
	return &Client{
		client:     client,
		api:        client.API(),
		dispatcher: dispatcher,
	}, nil
}

func (c *Client) Start(ctx context.Context, botToken string) error {
	return c.client.Run(ctx, func(ctx context.Context) error {
		status, err := c.client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("auth status failed: %w", err)
		}

		if !status.Authorized {
			if _, err := c.client.Auth().Bot(ctx, botToken); err != nil {
				return fmt.Errorf("bot login failed: %w", err)
			}
		}

		me, err := c.client.Self(ctx)
		if err != nil {
			return fmt.Errorf("get self failed: %w", err)
		}
		c.me = me

		logger.Info("Telegram client connected", "username", me.Username, "id", me.ID)
		
		<-ctx.Done()
		return nil
	})
}

func (c *Client) API() *tg.Client {
	return c.api
}

func (c *Client) Me() *tg.User {
	return c.me
}
