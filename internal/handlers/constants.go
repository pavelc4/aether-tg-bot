package handlers

import (
	"regexp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var commandHandlers = map[string]func(*tgbotapi.BotAPI, *tgbotapi.Message){
	"start":     handleStart,
	"help":      handleHelp,
	"speedtest": handleSpeedTest,
	"stats":     handleStats,
	"mp":        handleDownloadAudio,
	"video":     handleDownloadVideo,
	"dl":        handleDownloadGeneric,
}

var (
	urlRegex = regexp.MustCompile(`(https?://[^\s]+)`)
)

var platformMap = map[string]string{
	"instagram.com":   "Instagram",
	"tiktok.com":      "TikTok",
	"youtube.com":     "YouTube",
	"youtu.be":        "YouTube",
	"x.com":           "X",
	"facebook.com":    "Facebook",
	"fb.watch":        "Facebook",
	"reddit.com":      "Reddit",
	"pinterest.com":   "Pinterest",
	"soundcloud.com":  "SoundCloud",
	"vimeo.com":       "Vimeo",
	"dailymotion.com": "Dailymotion",
	"twitch.tv":       "Twitch",
	"bilibili.com":    "Bilibili",
	"snapchat.com":    "Snapchat",
	"tumblr.com":      "Tumblr",
	"ok.ru":           "OK.ru",
	"vk.com":          "VK",
}
