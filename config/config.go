package config

import (
	"os"
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

func GetYtdlpAPI() string {
	ytdlpAPI := os.Getenv("YTDLP_API")
	if ytdlpAPI == "" {
		ytdlpAPI = "http://yt-dlp-api:8080"
	}
	return ytdlpAPI
}
func GetTelegramApiURL() string {
	return os.Getenv("TELEGRAM_API_URL")
}
