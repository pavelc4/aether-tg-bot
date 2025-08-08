package config

import "os"

func GetBotToken() string {
	return os.Getenv("BOT_TOKEN")
}
