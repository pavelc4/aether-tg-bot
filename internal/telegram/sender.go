package telegram

import (
	"context"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/handlers"
)

const (
	updateTimeout     = 60
	workerPoolSize    = 100 // Limit concurrent goroutines
	shutdownTimeout   = 30 * time.Second
	processingTimeout = 10 * time.Minute // Max time for processing one update
)

// GetBotClient returns HTTP client untuk Telegram Bot API
func GetBotClient() *http.Client {
	return &http.Client{
		Timeout: 90 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			DisableKeepAlives:     false,
		},
	}
}

// StartBot initializes and runs the Telegram bot
func StartBot(token string) error {
	apiURL := config.GetTelegramApiURL()
	if apiURL == "" {
		apiURL = "http://localhost:8081"
		log.Printf("Using default Telegram API URL: %s", apiURL)
	}

	httpClient := GetBotClient()

	bot, err := tgbotapi.NewBotAPIWithClient(token, apiURL+"/bot%s/%s", httpClient)
	if err != nil {
		return err
	}

	log.Printf("🤖 Bot @%s is now online!", bot.Self.UserName)

	// Setup update configuration
	u := tgbotapi.NewUpdate(0)
	u.Timeout = updateTimeout

	updates := bot.GetUpdatesChan(u)

	// Worker pool to limit concurrent processing
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Semaphore to limit concurrent goroutines
	sem := make(chan struct{}, workerPoolSize)

	// Process updates
	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Acquire semaphore
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			log.Println("🛑 Bot shutting down...")
			return nil
		}

		// Process update in goroutine
		go func(update tgbotapi.Update) {
			defer func() {
				// Release semaphore
				<-sem

				// Recover from panic
				if r := recover(); r != nil {
					log.Printf("💥 Panic recovered in update handler: %v", r)
				}
			}()

			// Process with timeout context
			processCtx, processCancel := context.WithTimeout(ctx, processingTimeout)
			defer processCancel()

			processUpdate(processCtx, bot, &update)
		}(update)
	}

	return nil
}

// processUpdate handles a single update with context
func processUpdate(ctx context.Context, bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		log.Printf("⚠️  Update processing cancelled: %v", ctx.Err())
		return
	default:
	}

	msg := update.Message

	// Ignore old messages (older than 5 minutes)
	if time.Since(time.Unix(int64(msg.Date), 0)) > 5*time.Minute {
		log.Printf("⏭️  Ignoring old message from %s", msg.From.UserName)
		return
	}

	// Route to appropriate handler
	if msg.IsCommand() {
		handlers.HandleCommand(bot, msg)
	} else {
		handlers.HandleMessage(bot, msg)
	}
}

// GracefulShutdown handles bot shutdown with cleanup
func GracefulShutdown(bot *tgbotapi.BotAPI) {
	log.Println("🛑 Initiating graceful shutdown...")

	// Stop receiving updates
	bot.StopReceivingUpdates()

	// Wait for ongoing operations
	time.Sleep(shutdownTimeout)

	log.Println("✅ Bot shutdown complete")
}
