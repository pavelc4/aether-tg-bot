package main

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/pavelc4/aether-tg-bot/internal/app"
	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if memLimit := os.Getenv("GOMEMLIMIT"); memLimit == "" {
		if limit := runtime.MemProfileRate; limit > 0 {
			runtime.MemProfileRate = 512 * 1024
		}
	}

	logger.Info("GC optimized", "procs", runtime.NumCPU())
}

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
