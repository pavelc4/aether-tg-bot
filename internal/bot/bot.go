package bot

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/pavelc4/aether-tg-bot/internal/bot/handlers"
	"github.com/pavelc4/aether-tg-bot/internal/stats"
)

func StartBot() {
	// Initialize stats
	stats.InitStats()

	// Parse configuration
	apiIDStr := os.Getenv("TELEGRAM_API_ID")
	apiHash := os.Getenv("TELEGRAM_API_HASH")
	botToken := os.Getenv("BOT_TOKEN")

	if apiIDStr == "" || apiHash == "" || botToken == "" {
		log.Fatal("TELEGRAM_API_ID, TELEGRAM_API_HASH, or BOT_TOKEN is not set")
	}

	apiID, err := strconv.Atoi(apiIDStr)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_API_ID: %v", err)
	}

	// Create dispatcher
	dispatcher := tg.NewUpdateDispatcher()

	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Create client
	client := telegram.NewClient(apiID, apiHash, telegram.Options{
		UpdateHandler: dispatcher,
		SessionStorage: &session.FileStorage{
			Path: filepath.Join("data", "gotd_session.json"),
		},
	})

	// Register handlers
	h := handlers.NewHandler(client)
	h.Register(dispatcher)

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Starting Aether Bot (Gotd Edition)...")

	if err := client.Run(ctx, func(ctx context.Context) error {
		// Authenticate
		status, err := client.Auth().Status(ctx)
		if err != nil {
			return err
		}

		if !status.Authorized {
			if _, err := client.Auth().Bot(ctx, botToken); err != nil {
				return err
			}
		}

		// Get self info
		me, err := client.Self(ctx)
		if err != nil {
			return err
		}

		log.Printf("Bot @%s is now online!", me.Username)

		// Wait for context cancellation
		<-ctx.Done()
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	log.Println("Bot stopped.")
}
