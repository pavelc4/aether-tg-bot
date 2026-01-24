package app

import (
	"context"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/bot"
	"github.com/pavelc4/aether-tg-bot/internal/handler"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

type App struct {
	Bot *bot.Bot
	Cfg *config.Config
}

func New() (*App, error) {

	cfg := config.LoadConfig()

	provider.Register(provider.NewCobalt())
	provider.Register(provider.NewYouTube())
	provider.Register(provider.NewTikTok())


	streamMgr := streaming.NewManager(streaming.Config{
		MaxConcurrentStreams: config.DefaultMaxConcurrentStreams,
		UploadWorkers:        config.DefaultUploadWorkers,
		BufferSize:           config.DefaultBufferSize,
		ChunkSize:            config.DefaultChunkSize,
		RetryLimit:           config.DefaultRetryLimit,
	})


	dispatcher := tg.NewUpdateDispatcher()
	
	client, err := telegram.NewClient(cfg, dispatcher)
	if err != nil {
		return nil, err
	}

	dlHandler := handler.NewDownloadHandler(streamMgr, client)
	adminHandler := handler.NewAdminHandler(client)
	
	router := bot.NewRouter(dlHandler, adminHandler)
	
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		return router.OnMessage(ctx, e, update)
	})
	
	b := bot.New(client, router)
	
	logger.Info("Application initialized successfully")
	return &App{
		Bot: b,
		Cfg: cfg,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	return a.Bot.Run(ctx, a.Cfg.BotToken)
}
