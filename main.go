package main

import (
	"log"

	"github.com/pavelc4/aether-tg-bot/bot"
	"github.com/pavelc4/aether-tg-bot/config"
)

func main() {
	token := config.GetBotToken()
	if token == "" {
		log.Fatal("BOT_TOKEN is not set")
	}

	if err := bot.StartBot(token); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
}
