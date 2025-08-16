package main

import (
	"log"
	"os"

	"github.com/pavelc4/aether-tg-bot/bot"
)

func GetBotToken() string {
	return os.Getenv("BOT_TOKEN")
}

func GetCobaltAPI() string {
	cobaltAPI := os.Getenv("COBALT_API")
	if cobaltAPI == "" {
		cobaltAPI = "http://cobalt:9000"
	}
	return cobaltAPI
}
func GetTelegramApiURL() string {
	return os.Getenv("TELEGRAM_API_URL")
}

func main() {
	token := GetBotToken()
	if token == "" {
		log.Fatal("BOT_TOKEN is not set")
	}

	if err := bot.StartBot(token); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
}
