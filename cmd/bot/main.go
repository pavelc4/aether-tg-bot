package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/bot"
)

func main() {
	token := config.GetBotToken()
	if token == "" {
		log.Fatal(" BOT_TOKEN is not set")
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := bot.StartBot(token); err != nil {
			log.Fatalf("Failed to start bot: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down...")
}
