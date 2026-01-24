package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/pavelc4/aether-tg-bot/internal/app"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

func main() {

	application, err := app.New()
	if err != nil {
		logger.Error("Failed to initialize app", "error", err)
		os.Exit(1)
	}


	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logger.Info("Starting Aether Bot...")

	if err := application.Start(ctx); err != nil {
		if ctx.Err() == nil {
			logger.Error("Bot stopped with error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Bot stopped gracefully")
}
