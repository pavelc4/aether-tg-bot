package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
)

const (
	// Default values
	defaultCobaltAPI   = "http://cobalt:9000"
	defaultYtdlpAPI    = "http://yt-dlp-api:8080"
	defaultTelegramAPI = "http://localhost:8081"
)

var (
	// Cache for config values with sync.Once
	botToken     string
	botTokenOnce sync.Once

	cobaltAPI     string
	cobaltAPIOnce sync.Once

	ytdlpAPI     string
	ytdlpAPIOnce sync.Once

	telegramAPI     string
	telegramAPIOnce sync.Once

	ownerID     int64
	ownerIDOnce sync.Once
)

// GetBotToken returns bot token with caching
func GetBotToken() string {
	botTokenOnce.Do(func() {
		botToken = os.Getenv("BOT_TOKEN")
		if botToken == "" {
			log.Fatal("BOT_TOKEN environment variable is required")
		}
	})
	return botToken
}

// GetCobaltAPI returns Cobalt API URL with caching and default
func GetCobaltAPI() string {
	cobaltAPIOnce.Do(func() {
		cobaltAPI = getEnvWithDefault("COBALT_API", defaultCobaltAPI)
		log.Printf("Using Cobalt API: %s", cobaltAPI)
	})
	return cobaltAPI
}

// GetYtdlpAPI returns yt-dlp API URL with caching and default
func GetYtdlpAPI() string {
	ytdlpAPIOnce.Do(func() {
		ytdlpAPI = getEnvWithDefault("YTDLP_API", defaultYtdlpAPI)
		log.Printf("Using yt-dlp API: %s", ytdlpAPI)
	})
	return ytdlpAPI
}

// GetTelegramApiURL returns Telegram API URL with caching and default
func GetTelegramApiURL() string {
	telegramAPIOnce.Do(func() {
		telegramAPI = getEnvWithDefault("TELEGRAM_API_URL", defaultTelegramAPI)
		log.Printf("Using Telegram API: %s", telegramAPI)
	})
	return telegramAPI
}

// GetOwnerID returns owner ID with caching and validation
func GetOwnerID() int64 {
	ownerIDOnce.Do(func() {
		ownerStr := os.Getenv("OWNER_ID")
		if ownerStr == "" {
			log.Println("Warning: OWNER_ID not set. Stats command will be disabled.")
			ownerID = 0
			return
		}

		id, err := strconv.ParseInt(ownerStr, 10, 64)
		if err != nil {
			log.Printf("Warning: Invalid OWNER_ID '%s': %v. Stats command will be disabled.", ownerStr, err)
			ownerID = 0
			return
		}

		ownerID = id
		log.Printf("Owner ID set to: %d", ownerID)
	})
	return ownerID
}

// Helper: Get environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Config struct for optional structured config
type Config struct {
	BotToken    string
	CobaltAPI   string
	YtdlpAPI    string
	TelegramAPI string
	OwnerID     int64
}

// LoadConfig loads all config at once (optional alternative approach)
func LoadConfig() *Config {
	return &Config{
		BotToken:    GetBotToken(),
		CobaltAPI:   GetCobaltAPI(),
		YtdlpAPI:    GetYtdlpAPI(),
		TelegramAPI: GetTelegramApiURL(),
		OwnerID:     GetOwnerID(),
	}
}

// ValidateConfig validates all required configurations
func ValidateConfig() error {
	if GetBotToken() == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	// Log configuration summary
	log.Println("Configuration loaded successfully:")
	log.Printf("  - Cobalt API: %s", GetCobaltAPI())
	log.Printf("  - yt-dlp API: %s", GetYtdlpAPI())
	log.Printf("  - Telegram API: %s", GetTelegramApiURL())
	if GetOwnerID() != 0 {
		log.Printf("  - Owner ID: %d", GetOwnerID())
	}

	return nil
}
