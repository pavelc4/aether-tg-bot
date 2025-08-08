package main

import (
	"log"

	"github.com/pavelc4/aether-tg-bot/bot"
	"github.com/pavelc4/aether-tg-bot/config"
)

func main() {
	token := config.GetBotToken()
	if token == "" {
		log.Fatalln("Bot token not found")
	}

	err := bot.StartBot(token)
	if err != nil {
		log.Fatalln(err)
	}
}
