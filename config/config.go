package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvBotToken          = "BOT_TOKEN"
	EnvCobaltAPI         = "COBALT_API"
	EnvCobaltAPIKey      = "COBALT_API_KEY"
	EnvYtdlpAPI          = "YTDLP_API"
	EnvYtdlpCookies      = "YTDLP_COOKIES"
	EnvTelegramAPI       = "TELEGRAM_API_URL"
	EnvOwnerID           = "OWNER_ID"
	EnvEnableAdaptive    = "ENABLE_ADAPTIVE_DOWNLOAD"
	EnvMaxFileSize       = "MAX_FILE_SIZE_MB"
	EnvUpdateTimeout     = "UPDATE_TIMEOUT"
	EnvWorkerPoolSize    = "WORKER_POOL_SIZE"
	EnvShutdownTimeout   = "SHUTDOWN_TIMEOUT_SECONDS"
	EnvProcessingTimeout = "PROCESSING_TIMEOUT_MINUTES"
)

const (
	DefaultCobaltAPI            = "http://cobalt:9000"
	DefaultYtdlpAPI             = "http://yt-dlp-api:8080"
	DefaultTelegramAPI          = "http://localhost:8081"
	DefaultMaxFileSize          = 50 // MB (Telegram limit)
	DefaultEnableAdaptive       = true
	DefaultUpdateTimeout        = 60
	DefaultWorkerPoolSize       = 100
	DefaultShutdownTimeout      = 30
	DefaultProcessingTimeout    = 10

	// Streaming Defaults
	DefaultMaxConcurrentStreams = 8
	DefaultChunkSize            = 512 * 1024 // 512KB
	DefaultBufferSize           = 5
	DefaultUploadWorkers        = 3
	DefaultRetryLimit           = 3
)

type Config struct {
	AppID      int
	AppHash    string
	BotToken   string
	SessionDir string
	TempDir    string
	CookiesDir string
	CobaltAPI         string
	CobaltAPIKey      string
	YtdlpAPI          string
	YtdlpCookies      string
	TelegramAPI       string
	OwnerID           int64
	EnableAdaptive    bool
	MaxFileSizeMB     int64
	MaxFileSizeBytes  int64
	UpdateTimeout     int
	WorkerPoolSize    int
	ShutdownTimeout   time.Duration
	ProcessingTimeout time.Duration
}

var currentConfig *Config

func init() {
	LoadConfig()
}

func LoadConfig() *Config {
	cfg := &Config{
		AppID:             getIntEnv("TELEGRAM_APP_ID", 0),
		AppHash:           os.Getenv("TELEGRAM_APP_HASH"),
		BotToken:          os.Getenv(EnvBotToken),
		SessionDir:        getEnvWithDefault("SESSION_DIR", "data"),
		TempDir:           getEnvWithDefault("TEMP_DIR", "tmp"),
		CookiesDir:        getEnvWithDefault("COOKIES_DIR", "cookies"),
		CobaltAPI:         getEnvWithDefault(EnvCobaltAPI, DefaultCobaltAPI),
		CobaltAPIKey:      os.Getenv(EnvCobaltAPIKey),
		YtdlpAPI:          getEnvWithDefault(EnvYtdlpAPI, DefaultYtdlpAPI),
		YtdlpCookies:      os.Getenv(EnvYtdlpCookies),
		TelegramAPI:       getEnvWithDefault(EnvTelegramAPI, DefaultTelegramAPI),
		EnableAdaptive:    getBoolEnv(EnvEnableAdaptive, DefaultEnableAdaptive),
		UpdateTimeout:     getIntEnv(EnvUpdateTimeout, DefaultUpdateTimeout),
		WorkerPoolSize:    getIntEnv(EnvWorkerPoolSize, DefaultWorkerPoolSize),
		ShutdownTimeout:   getDurationEnv(EnvShutdownTimeout, DefaultShutdownTimeout, time.Second),
		ProcessingTimeout: getDurationEnv(EnvProcessingTimeout, DefaultProcessingTimeout, time.Minute),
	}

	// Owner ID
	if ownerStr := os.Getenv(EnvOwnerID); ownerStr != "" {
		if id, err := strconv.ParseInt(ownerStr, 10, 64); err == nil {
			cfg.OwnerID = id
		} else {
			log.Printf("Invalid OWNER_ID '%s': %v", ownerStr, err)
		}
	}

	// Max File Size
	if sizeStr := os.Getenv(EnvMaxFileSize); sizeStr != "" {
		if sizeMB, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && sizeMB > 0 {
			cfg.MaxFileSizeMB = sizeMB
		} else {
			cfg.MaxFileSizeMB = DefaultMaxFileSize
		}
	} else {
		cfg.MaxFileSizeMB = DefaultMaxFileSize
	}
	cfg.MaxFileSizeBytes = cfg.MaxFileSizeMB * 1024 * 1024

	if cfg.YtdlpCookies != "" {
		if _, err := os.Stat(cfg.YtdlpCookies); err != nil {
			log.Printf("Cookie file not found: %s", cfg.YtdlpCookies)
			cfg.YtdlpCookies = ""
		}
	}

	currentConfig = cfg
	return cfg
}

func GetBotToken() string {
	if currentConfig == nil {
		return os.Getenv(EnvBotToken)
	}
	return currentConfig.BotToken
}

func GetCobaltAPI() string {
	if currentConfig == nil {
		return DefaultCobaltAPI
	}
	return currentConfig.CobaltAPI
}

func GetCobaltAPIKey() string {
	if currentConfig == nil {
		return ""
	}
	return currentConfig.CobaltAPIKey
}

func GetYtdlpAPI() string {
	if currentConfig == nil {
		return DefaultYtdlpAPI
	}
	return currentConfig.YtdlpAPI
}

func GetYtdlpCookies() string {
	if currentConfig == nil {
		return ""
	}
	return currentConfig.YtdlpCookies
}

func GetTelegramApiURL() string {
	if currentConfig == nil {
		return DefaultTelegramAPI
	}
	return currentConfig.TelegramAPI
}

func GetOwnerID() int64 {
	if currentConfig == nil {
		return 0
	}
	return currentConfig.OwnerID
}

func GetEnableAdaptive() bool {
	if currentConfig == nil {
		return DefaultEnableAdaptive
	}
	return currentConfig.EnableAdaptive
}

func GetMaxFileSize() int64 {
	if currentConfig == nil {
		return DefaultMaxFileSize * 1024 * 1024
	}
	return currentConfig.MaxFileSizeBytes
}

func GetUpdateTimeout() int {
	if currentConfig == nil {
		return DefaultUpdateTimeout
	}
	return currentConfig.UpdateTimeout
}

func GetWorkerPoolSize() int {
	if currentConfig == nil {
		return DefaultWorkerPoolSize
	}
	return currentConfig.WorkerPoolSize
}

func GetShutdownTimeout() time.Duration {
	if currentConfig == nil {
		return time.Duration(DefaultShutdownTimeout) * time.Second
	}
	return currentConfig.ShutdownTimeout
}

func GetProcessingTimeout() time.Duration {
	if currentConfig == nil {
		return time.Duration(DefaultProcessingTimeout) * time.Minute
	}
	return currentConfig.ProcessingTimeout
}

func ValidateConfig() error {
	log.Println("🔍 Validating configuration...")

	if GetBotToken() == "" {
		return fmt.Errorf("BOT_TOKEN is required")
	}

	if err := validateURL(GetCobaltAPI(), "COBALT_API"); err != nil {
		return err
	}

	if err := validateURL(GetYtdlpAPI(), "YTDLP_API"); err != nil {
		return err
	}

	if err := validateURL(GetTelegramApiURL(), "TELEGRAM_API_URL"); err != nil {
		return err
	}

	log.Println(" Configuration loaded successfully:")
	PrintConfig()

	return nil
}

func PrintConfig() {
	cfg := currentConfig
	if cfg == nil {
		cfg = LoadConfig()
	}

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
	log.Printf("  Update Timeout: %d seconds", cfg.UpdateTimeout)
	log.Printf("  Worker Pool Size: %d", cfg.WorkerPoolSize)
	log.Printf("  Shutdown Timeout: %v", cfg.ShutdownTimeout)
	log.Printf("  Processing Timeout: %v", cfg.ProcessingTimeout)
}
func ReloadConfig() {
	log.Println("🔄 Reloading configuration...")
	LoadConfig()
	time.Sleep(100 * time.Millisecond)
	log.Println(" Configuration reloaded")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil && val > 0 {
			return val
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if valStr := os.Getenv(key); valStr != "" {
		return valStr == "true" || valStr == "1"
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue int, unit time.Duration) time.Duration {
	if valStr := os.Getenv(key); valStr != "" {
		if val, err := strconv.Atoi(valStr); err == nil && val > 0 {
			return time.Duration(val) * unit
		}
	}
	return time.Duration(defaultValue) * unit
}

func validateURL(urlStr, name string) error {
	if urlStr == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return fmt.Errorf("%s must start with http:// or https://: %s", name, urlStr)
	}

	return nil
}

func maskToken(token string) string {
	if token == "" {
		return "not set"
	}

	if len(token) <= 8 {
		return "***"
	}

	return token[:4] + "..." + token[len(token)-4:]
}
