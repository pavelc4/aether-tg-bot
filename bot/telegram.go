package bot

import (
	"log"
	"net/http"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pavelc4/aether-tg-bot/config"
)

func StartBot(token string) error {
	apiURL := config.GetTelegramApiURL()

	if apiURL == "" {
		apiURL = "http://localhost:8081"
	}

	bot, err := tgbotapi.NewBotAPIWithClient(token, apiURL+"/bot%s/%s", &http.Client{})
	if err != nil {
		return err
	}

	log.Printf("Bot %s sudah online!", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			handleCommand(bot, update.Message)
		} else {
			handleMessage(bot, update.Message)
		}
	}
	return nil
}
