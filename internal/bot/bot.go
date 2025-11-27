package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/bot/handlers"
	pkghttp "github.com/pavelc4/aether-tg-bot/pkg/http"
)

func GetBotClient() *http.Client {
	return pkghttp.GetBotClient()
}

func StartBot(token string) error {
	apiURL := config.GetTelegramApiURL()
	if apiURL == "" {
		apiURL = "http://localhost:8081"
		log.Printf("Using default Telegram API URL: %s", apiURL)
	}

	httpClient := GetBotClient()

	bot, err := tgbotapi.NewBotAPIWithClient(token, apiURL+"/bot%s/%s", httpClient)
	if err != nil {
		return fmt.Errorf("failed to create bot API client: %w", err)
	}

	log.Printf(" Bot @%s is now online!", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = config.GetUpdateTimeout()

	updates := bot.GetUpdatesChan(u)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := make(chan struct{}, config.GetWorkerPoolSize())

	for update := range updates {
		if update.Message == nil {
			continue
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			log.Println(" Bot shutting down...")
			return nil
		}

		go func(update tgbotapi.Update) {
			defer func() {
				<-sem

				if r := recover(); r != nil {
					log.Printf(" Panic recovered in update handler: %v", r)
				}
			}()

			processCtx, processCancel := context.WithTimeout(ctx, config.GetProcessingTimeout())
			defer processCancel()

			processUpdate(processCtx, bot, &update)
		}(update)
	}

	return nil
}

func processUpdate(ctx context.Context, bot *tgbotapi.BotAPI, update *tgbotapi.Update) {
	select {
	case <-ctx.Done():
		log.Printf("  Update processing cancelled: %v", ctx.Err())
		return
	default:
	}

	msg := update.Message

	if time.Since(time.Unix(int64(msg.Date), 0)) > 5*time.Minute {
		log.Printf("  Ignoring old message from %s", msg.From.UserName)
		return
	}

	if msg.IsCommand() {
		handlers.HandleCommand(bot, msg)
	} else {
		handlers.HandleMessage(bot, msg)
	}
}

func GracefulShutdown(bot *tgbotapi.BotAPI) {
	log.Println(" Initiating graceful shutdown...")

	bot.StopReceivingUpdates()

	time.Sleep(config.GetShutdownTimeout())

	log.Println(" Bot shutdown complete")
}
