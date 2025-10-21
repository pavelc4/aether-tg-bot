package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pavelc4/aether-tg-bot/config"
	"github.com/pavelc4/aether-tg-bot/internal/telegram"
)

func main() {
	token := config.GetBotToken()
	if token == "" {
		log.Fatal("❌ BOT_TOKEN is not set")
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start bot
	go func() {
		if err := telegram.StartBot(token); err != nil {
			log.Fatalf("❌ Bot error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("👋 Shutting down...")
}
