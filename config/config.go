package config

import "os"

func GetBotToken() string {
	return os.Getenv("BOT_TOKEN")
}

func GetCobaltAPI() string {

	cobaltAPI := os.Getenv("COBALT_API")
	if cobaltAPI == "" {
		cobaltAPI = "http://127.10.10.1:8080"
	}
	return cobaltAPI
}
