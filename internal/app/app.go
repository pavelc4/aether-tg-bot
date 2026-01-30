package app

import (
	"context"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/bot"
	"github.com/pavelc4/aether-tg-bot/internal/handler"
	"github.com/pavelc4/aether-tg-bot/internal/middleware"
	"github.com/pavelc4/aether-tg-bot/internal/provider"
	"github.com/pavelc4/aether-tg-bot/internal/streaming"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
	"runtime"
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


	maxStreams := cfg.MaxConcurrentStreams
	if maxStreams <= 0 {
		maxStreams = runtime.NumCPU() * 4
		if maxStreams < config.DefaultMaxConcurrentStreams {
			maxStreams = config.DefaultMaxConcurrentStreams
		}
		logger.Info("Using adaptive concurrency", "cores", runtime.NumCPU(), "limit", maxStreams)
	} else {
		logger.Info("Using fixed concurrency", "limit", maxStreams)
	}

	streamMgr := streaming.NewManager(streaming.Config{
		MaxConcurrentStreams: maxStreams,
		MinUploadWorkers:     cfg.MinUploadWorkers,
		MaxUploadWorkers:     cfg.MaxUploadWorkers,
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
	adminHandler := handler.NewAdminHandler(client, streamMgr)
	basicHandler := handler.NewBasicHandler(client)
	
	router := bot.NewRouter(dlHandler, adminHandler, basicHandler)
	
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		handler := func() {
			if err := router.OnMessage(ctx, e, update); err != nil {
				logger.Error("OnMessage failed", "error", err)
			}
		}
		go middleware.Chain(handler, 
			middleware.Recover,
			func(next func()) func() { return middleware.Logger("OnNewMessage", next) },
		)()
		return nil
	})

	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		handler := func() {
			if err := router.OnChannelMessage(ctx, e, update); err != nil {
				logger.Error("OnChannelMessage failed", "error", err)
			}
		}
		go middleware.Chain(handler, 
			middleware.Recover,
			func(next func()) func() { return middleware.Logger("OnNewChannelMessage", next) },
		)()
		return nil
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
