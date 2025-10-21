package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

// Environment variable keys
const (
	EnvBotToken       = "BOT_TOKEN"
	EnvCobaltAPI      = "COBALT_API"
	EnvCobaltAPIKey   = "COBALT_API_KEY"
	EnvYtdlpAPI       = "YTDLP_API"
	EnvYtdlpCookies   = "YTDLP_COOKIES"
	EnvTelegramAPI    = "TELEGRAM_API_URL"
	EnvOwnerID        = "OWNER_ID"
	EnvEnableAdaptive = "ENABLE_ADAPTIVE_DOWNLOAD"
	EnvMaxFileSize    = "MAX_FILE_SIZE_MB"
)

// Default values
const (
	DefaultCobaltAPI      = "http://cobalt:9000"
	DefaultYtdlpAPI       = "http://yt-dlp-api:8080"
	DefaultTelegramAPI    = "http://localhost:8081"
	DefaultMaxFileSize    = 50 // MB (Telegram limit)
	DefaultEnableAdaptive = true
)

var (
	// Cached config values with sync.Once
	botToken     string
	botTokenOnce sync.Once

	cobaltAPI     string
	cobaltAPIOnce sync.Once

	cobaltAPIKey     string
	cobaltAPIKeyOnce sync.Once

	ytdlpAPI     string
	ytdlpAPIOnce sync.Once

	ytdlpCookies     string
	ytdlpCookiesOnce sync.Once

	telegramAPI     string
	telegramAPIOnce sync.Once

	ownerID     int64
	ownerIDOnce sync.Once

	enableAdaptive     bool
	enableAdaptiveOnce sync.Once

	maxFileSize     int64
	maxFileSizeOnce sync.Once
)

// Config struct for structured configuration
type Config struct {
	BotToken         string
	CobaltAPI        string
	CobaltAPIKey     string
	YtdlpAPI         string
	YtdlpCookies     string
	TelegramAPI      string
	OwnerID          int64
	EnableAdaptive   bool
	MaxFileSizeMB    int64
	MaxFileSizeBytes int64
}

// GetBotToken returns bot token with caching
func GetBotToken() string {
	botTokenOnce.Do(func() {
		botToken = os.Getenv(EnvBotToken)
		if botToken == "" {
			log.Fatal("❌ BOT_TOKEN environment variable is required")
		}
	})
	return botToken
}

// GetCobaltAPI returns Cobalt API URL with caching and default
func GetCobaltAPI() string {
	cobaltAPIOnce.Do(func() {
		cobaltAPI = getEnvWithDefault(EnvCobaltAPI, DefaultCobaltAPI)
		log.Printf("🔗 Cobalt API: %s", cobaltAPI)
	})
	return cobaltAPI
}

// GetCobaltAPIKey returns Cobalt API key (optional)
func GetCobaltAPIKey() string {
	cobaltAPIKeyOnce.Do(func() {
		cobaltAPIKey = os.Getenv(EnvCobaltAPIKey)
		if cobaltAPIKey != "" {
			log.Printf("🔑 Cobalt API Key: configured")
		}
	})
	return cobaltAPIKey
}

// GetYtdlpAPI returns yt-dlp API URL with caching and default
func GetYtdlpAPI() string {
	ytdlpAPIOnce.Do(func() {
		ytdlpAPI = getEnvWithDefault(EnvYtdlpAPI, DefaultYtdlpAPI)
		log.Printf("🔗 yt-dlp API: %s", ytdlpAPI)
	})
	return ytdlpAPI
}

// GetYtdlpCookies returns path to yt-dlp cookies file (optional)
func GetYtdlpCookies() string {
	ytdlpCookiesOnce.Do(func() {
		ytdlpCookies = os.Getenv(EnvYtdlpCookies)
		if ytdlpCookies != "" {
			if _, err := os.Stat(ytdlpCookies); err == nil {
				log.Printf("🍪 yt-dlp Cookies: %s", ytdlpCookies)
			} else {
				log.Printf("⚠️  Cookie file not found: %s", ytdlpCookies)
				ytdlpCookies = ""
			}
		}
	})
	return ytdlpCookies
}

// GetTelegramApiURL returns Telegram API URL with caching and default
func GetTelegramApiURL() string {
	telegramAPIOnce.Do(func() {
		telegramAPI = getEnvWithDefault(EnvTelegramAPI, DefaultTelegramAPI)
		log.Printf("🔗 Telegram API: %s", telegramAPI)
	})
	return telegramAPI
}

// GetOwnerID returns owner ID with caching and validation
func GetOwnerID() int64 {
	ownerIDOnce.Do(func() {
		ownerStr := os.Getenv(EnvOwnerID)
		if ownerStr == "" {
			log.Println("⚠️  OWNER_ID not set. Admin commands will be disabled.")
			ownerID = 0
			return
		}

		id, err := strconv.ParseInt(ownerStr, 10, 64)
		if err != nil {
			log.Printf("⚠️  Invalid OWNER_ID '%s': %v. Admin commands will be disabled.", ownerStr, err)
			ownerID = 0
			return
		}

		ownerID = id
		log.Printf("👤 Owner ID: %d", ownerID)
	})
	return ownerID
}

// GetEnableAdaptive returns whether adaptive download is enabled
func GetEnableAdaptive() bool {
	enableAdaptiveOnce.Do(func() {
		envVal := os.Getenv(EnvEnableAdaptive)
		if envVal == "" {
			enableAdaptive = DefaultEnableAdaptive
		} else {
			enableAdaptive = envVal == "true" || envVal == "1"
		}
		log.Printf("⚡ Adaptive Download: %v", enableAdaptive)
	})
	return enableAdaptive
}

// GetMaxFileSize returns max file size in bytes
func GetMaxFileSize() int64 {
	maxFileSizeOnce.Do(func() {
		envVal := os.Getenv(EnvMaxFileSize)
		if envVal == "" {
			maxFileSize = DefaultMaxFileSize * 1024 * 1024
		} else {
			sizeMB, err := strconv.ParseInt(envVal, 10, 64)
			if err != nil || sizeMB <= 0 {
				log.Printf("⚠️  Invalid MAX_FILE_SIZE_MB '%s', using default: %d MB", envVal, DefaultMaxFileSize)
				maxFileSize = DefaultMaxFileSize * 1024 * 1024
			} else {
				maxFileSize = sizeMB * 1024 * 1024
			}
		}
		log.Printf("📦 Max File Size: %d MB", maxFileSize/(1024*1024))
	})
	return maxFileSize
}

// LoadConfig loads all config at once (alternative approach)
func LoadConfig() *Config {
	return &Config{
		BotToken:         GetBotToken(),
		CobaltAPI:        GetCobaltAPI(),
		CobaltAPIKey:     GetCobaltAPIKey(),
		YtdlpAPI:         GetYtdlpAPI(),
		YtdlpCookies:     GetYtdlpCookies(),
		TelegramAPI:      GetTelegramApiURL(),
		OwnerID:          GetOwnerID(),
		EnableAdaptive:   GetEnableAdaptive(),
		MaxFileSizeMB:    GetMaxFileSize() / (1024 * 1024),
		MaxFileSizeBytes: GetMaxFileSize(),
	}
}

// ValidateConfig validates all required configurations
func ValidateConfig() error {
	log.Println("🔍 Validating configuration...")

	// Check required configs
	if GetBotToken() == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	// Validate URLs (basic check)
	if err := validateURL(GetCobaltAPI(), "COBALT_API"); err != nil {
		return err
	}

	if err := validateURL(GetYtdlpAPI(), "YTDLP_API"); err != nil {
		return err
	}

	if err := validateURL(GetTelegramApiURL(), "TELEGRAM_API_URL"); err != nil {
		return err
	}

	// Log configuration summary
	log.Println("✅ Configuration loaded successfully:")
	log.Printf("  📍 Cobalt API: %s", GetCobaltAPI())
	log.Printf("  📍 yt-dlp API: %s", GetYtdlpAPI())
	log.Printf("  📍 Telegram API: %s", GetTelegramApiURL())

	if GetCobaltAPIKey() != "" {
		log.Printf("  🔑 Cobalt API Key: configured")
	}

	if GetYtdlpCookies() != "" {
		log.Printf("  🍪 yt-dlp Cookies: %s", GetYtdlpCookies())
	}

	if GetOwnerID() != 0 {
		log.Printf("  👤 Owner ID: %d", GetOwnerID())
	}

	log.Printf("  ⚡ Adaptive Download: %v", GetEnableAdaptive())
	log.Printf("  📦 Max File Size: %d MB", GetMaxFileSize()/(1024*1024))

	return nil
}

// PrintConfig prints current configuration (for debugging)
func PrintConfig() {
	cfg := LoadConfig()

	log.Println("📋 Current Configuration:")
	log.Printf("  Bot Token: %s", maskToken(cfg.BotToken))
	log.Printf("  Cobalt API: %s", cfg.CobaltAPI)
	log.Printf("  Cobalt API Key: %s", maskToken(cfg.CobaltAPIKey))
	log.Printf("  yt-dlp API: %s", cfg.YtdlpAPI)
	log.Printf("  yt-dlp Cookies: %s", cfg.YtdlpCookies)
	log.Printf("  Telegram API: %s", cfg.TelegramAPI)
	log.Printf("  Owner ID: %d", cfg.OwnerID)
	log.Printf("  Adaptive Download: %v", cfg.EnableAdaptive)
	log.Printf("  Max File Size: %d MB", cfg.MaxFileSizeMB)
}

// Helper functions

// getEnvWithDefault gets environment variable with default value
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// validateURL performs basic URL validation
func validateURL(urlStr, name string) error {
	if urlStr == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}

	// Basic validation - check if it starts with http:// or https://
	if urlStr[:4] != "http" {
		return fmt.Errorf("%s must start with http:// or https://: %s", name, urlStr)
	}

	return nil
}

// maskToken masks sensitive token for logging
func maskToken(token string) string {
	if token == "" {
		return "not set"
	}

	if len(token) <= 8 {
		return "***"
	}

	return token[:4] + "..." + token[len(token)-4:]
}

// ReloadConfig forces reload of all cached config (for testing)
func ReloadConfig() {
	log.Println("🔄 Reloading configuration...")

	// Reset all sync.Once
	botTokenOnce = sync.Once{}
	cobaltAPIOnce = sync.Once{}
	cobaltAPIKeyOnce = sync.Once{}
	ytdlpAPIOnce = sync.Once{}
	ytdlpCookiesOnce = sync.Once{}
	telegramAPIOnce = sync.Once{}
	ownerIDOnce = sync.Once{}
	enableAdaptiveOnce = sync.Once{}
	maxFileSizeOnce = sync.Once{}

	// Wait a bit for any pending operations
	time.Sleep(100 * time.Millisecond)

	log.Println("✅ Configuration reloaded")
}
