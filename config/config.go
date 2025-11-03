package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
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
	DefaultCobaltAPI         = "http://cobalt:9000"
	DefaultYtdlpAPI          = "http://yt-dlp-api:8080"
	DefaultTelegramAPI       = "http://localhost:8081"
	DefaultMaxFileSize       = 50 // MB (Telegram limit)
	DefaultEnableAdaptive    = true
	DefaultUpdateTimeout     = 60
	DefaultWorkerPoolSize    = 100
	DefaultShutdownTimeout   = 30
	DefaultProcessingTimeout = 10
)

var (
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

	updateTimeout     int
	updateTimeoutOnce sync.Once

	workerPoolSize     int
	workerPoolSizeOnce sync.Once

	shutdownTimeout     time.Duration
	shutdownTimeoutOnce sync.Once

	processingTimeout     time.Duration
	processingTimeoutOnce sync.Once
)

type Config struct {
	BotToken          string
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

func GetBotToken() string {
	botTokenOnce.Do(func() {
		botToken = os.Getenv(EnvBotToken)
		if botToken == "" {
			log.Fatal(" BOT_TOKEN environment variable is required")
		}
	})
	return botToken
}

func GetCobaltAPI() string {
	cobaltAPIOnce.Do(func() {
		cobaltAPI = getEnvWithDefault(EnvCobaltAPI, DefaultCobaltAPI)
		log.Printf(" Cobalt API: %s", cobaltAPI)
	})
	return cobaltAPI
}

func GetCobaltAPIKey() string {
	cobaltAPIKeyOnce.Do(func() {
		cobaltAPIKey = os.Getenv(EnvCobaltAPIKey)
		if cobaltAPIKey != "" {
			log.Printf(" Cobalt API Key: configured")
		}
	})
	return cobaltAPIKey
}

func GetYtdlpAPI() string {
	ytdlpAPIOnce.Do(func() {
		ytdlpAPI = getEnvWithDefault(EnvYtdlpAPI, DefaultYtdlpAPI)
		log.Printf(" yt-dlp API: %s", ytdlpAPI)
	})
	return ytdlpAPI
}

func GetYtdlpCookies() string {
	ytdlpCookiesOnce.Do(func() {
		ytdlpCookies = os.Getenv(EnvYtdlpCookies)
		if ytdlpCookies != "" {
			if _, err := os.Stat(ytdlpCookies); err == nil {
				log.Printf(" yt-dlp Cookies: %s", ytdlpCookies)
			} else {
				log.Printf("  Cookie file not found: %s", ytdlpCookies)
				ytdlpCookies = ""
			}
		}
	})
	return ytdlpCookies
}

func GetTelegramApiURL() string {
	telegramAPIOnce.Do(func() {
		telegramAPI = getEnvWithDefault(EnvTelegramAPI, DefaultTelegramAPI)
		log.Printf(" Telegram API: %s", telegramAPI)
	})
	return telegramAPI
}

func GetOwnerID() int64 {
	ownerIDOnce.Do(func() {
		ownerStr := os.Getenv(EnvOwnerID)
		if ownerStr == "" {
			log.Println("  OWNER_ID not set. Admin commands will be disabled.")
			ownerID = 0
			return
		}

		id, err := strconv.ParseInt(ownerStr, 10, 64)
		if err != nil {
			log.Printf("  Invalid OWNER_ID '%s': %v. Admin commands will be disabled.", ownerStr, err)
			ownerID = 0
			return
		}

		ownerID = id
		log.Printf(" Owner ID: %d", ownerID)
	})
	return ownerID
}

func GetEnableAdaptive() bool {
	enableAdaptiveOnce.Do(func() {
		envVal := os.Getenv(EnvEnableAdaptive)
		if envVal == "" {
			enableAdaptive = DefaultEnableAdaptive
		} else {
			enableAdaptive = envVal == "true" || envVal == "1"
		}
		log.Printf(" Adaptive Download: %v", enableAdaptive)
	})
	return enableAdaptive
}

func GetMaxFileSize() int64 {
	maxFileSizeOnce.Do(func() {
		envVal := os.Getenv(EnvMaxFileSize)
		if envVal == "" {
			maxFileSize = DefaultMaxFileSize * 1024 * 1024
		} else {
			sizeMB, err := strconv.ParseInt(envVal, 10, 64)
			if err != nil || sizeMB <= 0 {
				log.Printf("  Invalid MAX_FILE_SIZE_MB '%s', using default: %d MB", envVal, DefaultMaxFileSize)
				maxFileSize = DefaultMaxFileSize * 1024 * 1024
			} else {
				maxFileSize = sizeMB * 1024 * 1024
			}
		}
		log.Printf(" Max File Size: %d MB", maxFileSize/(1024*1024))
	})
	return maxFileSize
}

func GetUpdateTimeout() int {
	updateTimeoutOnce.Do(func() {
		envVal := os.Getenv(EnvUpdateTimeout)
		if envVal == "" {
			updateTimeout = DefaultUpdateTimeout
		} else {
			timeout, err := strconv.Atoi(envVal)
			if err != nil || timeout <= 0 {
				log.Printf("  Invalid UPDATE_TIMEOUT '%s', using default: %d", envVal, DefaultUpdateTimeout)
				updateTimeout = DefaultUpdateTimeout
			} else {
				updateTimeout = timeout
			}
		}
		log.Printf(" Update Timeout: %d seconds", updateTimeout)
	})
	return updateTimeout
}

func GetWorkerPoolSize() int {
	workerPoolSizeOnce.Do(func() {
		envVal := os.Getenv(EnvWorkerPoolSize)
		if envVal == "" {
			workerPoolSize = DefaultWorkerPoolSize
		} else {
			size, err := strconv.Atoi(envVal)
			if err != nil || size <= 0 {
				log.Printf("  Invalid WORKER_POOL_SIZE '%s', using default: %d", envVal, DefaultWorkerPoolSize)
				workerPoolSize = DefaultWorkerPoolSize
			} else {
				workerPoolSize = size
			}
		}
		log.Printf(" Worker Pool Size: %d", workerPoolSize)
	})
	return workerPoolSize
}

func GetShutdownTimeout() time.Duration {
	shutdownTimeoutOnce.Do(func() {
		envVal := os.Getenv(EnvShutdownTimeout)
		if envVal == "" {
			shutdownTimeout = time.Duration(DefaultShutdownTimeout) * time.Second
		} else {
			seconds, err := strconv.Atoi(envVal)
			if err != nil || seconds <= 0 {
				log.Printf("  Invalid SHUTDOWN_TIMEOUT_SECONDS '%s', using default: %d", envVal, DefaultShutdownTimeout)
				shutdownTimeout = time.Duration(DefaultShutdownTimeout) * time.Second
			} else {
				shutdownTimeout = time.Duration(seconds) * time.Second
			}
		}
		log.Printf(" Shutdown Timeout: %v", shutdownTimeout)
	})
	return shutdownTimeout
}

func GetProcessingTimeout() time.Duration {
	processingTimeoutOnce.Do(func() {
		envVal := os.Getenv(EnvProcessingTimeout)
		if envVal == "" {
			processingTimeout = time.Duration(DefaultProcessingTimeout) * time.Minute
		} else {
			minutes, err := strconv.Atoi(envVal)
			if err != nil || minutes <= 0 {
				log.Printf("  Invalid PROCESSING_TIMEOUT_MINUTES '%s', using default: %d", envVal, DefaultProcessingTimeout)
				processingTimeout = time.Duration(DefaultProcessingTimeout) * time.Minute
			} else {
				processingTimeout = time.Duration(minutes) * time.Minute
			}
		}
		log.Printf(" Processing Timeout: %v", processingTimeout)
	})
	return processingTimeout
}

func LoadConfig() *Config {
	return &Config{
		BotToken:          GetBotToken(),
		CobaltAPI:         GetCobaltAPI(),
		CobaltAPIKey:      GetCobaltAPIKey(),
		YtdlpAPI:          GetYtdlpAPI(),
		YtdlpCookies:      GetYtdlpCookies(),
		TelegramAPI:       GetTelegramApiURL(),
		OwnerID:           GetOwnerID(),
		EnableAdaptive:    GetEnableAdaptive(),
		MaxFileSizeMB:     GetMaxFileSize() / (1024 * 1024),
		MaxFileSizeBytes:  GetMaxFileSize(),
		UpdateTimeout:     GetUpdateTimeout(),
		WorkerPoolSize:    GetWorkerPoolSize(),
		ShutdownTimeout:   GetShutdownTimeout(),
		ProcessingTimeout: GetProcessingTimeout(),
	}
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
	log.Printf("   Cobalt API: %s", GetCobaltAPI())
	log.Printf("   yt-dlp API: %s", GetYtdlpAPI())
	log.Printf("   Telegram API: %s", GetTelegramApiURL())

	if GetCobaltAPIKey() != "" {
		log.Printf("   Cobalt API Key: configured")
	}

	if GetYtdlpCookies() != "" {
		log.Printf("   yt-dlp Cookies: %s", GetYtdlpCookies())
	}

	if GetOwnerID() != 0 {
		log.Printf("   Owner ID: %d", GetOwnerID())
	}

	log.Printf("   Adaptive Download: %v", GetEnableAdaptive())
	log.Printf("   Max File Size: %d MB", GetMaxFileSize()/(1024*1024))
	log.Printf("   Update Timeout: %d seconds", GetUpdateTimeout())
	log.Printf("   Worker Pool Size: %d", GetWorkerPoolSize())
	log.Printf("   Shutdown Timeout: %v", GetShutdownTimeout())
	log.Printf("   Processing Timeout: %v", GetProcessingTimeout())

	return nil
}

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
	log.Printf("  Update Timeout: %d seconds", cfg.UpdateTimeout)
	log.Printf("  Worker Pool Size: %d", cfg.WorkerPoolSize)
	log.Printf("  Shutdown Timeout: %v", cfg.ShutdownTimeout)
	log.Printf("  Processing Timeout: %v", cfg.ProcessingTimeout)
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func validateURL(urlStr, name string) error {
	if urlStr == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}

	if urlStr[:4] != "http" {
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

func ReloadConfig() {
	log.Println("🔄 Reloading configuration...")

	botTokenOnce = sync.Once{}
	cobaltAPIOnce = sync.Once{}
	cobaltAPIKeyOnce = sync.Once{}
	ytdlpAPIOnce = sync.Once{}
	ytdlpCookiesOnce = sync.Once{}
	telegramAPIOnce = sync.Once{}
	ownerIDOnce = sync.Once{}
	enableAdaptiveOnce = sync.Once{}
	maxFileSizeOnce = sync.Once{}
	updateTimeoutOnce = sync.Once{}
	workerPoolSizeOnce = sync.Once{}
	shutdownTimeoutOnce = sync.Once{}
	processingTimeoutOnce = sync.Once{}

	time.Sleep(100 * time.Millisecond)

	log.Println(" Configuration reloaded")
}
