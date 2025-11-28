package handlers

import "strings"

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

func DetectSource(url string) string {
	urlLower := strings.ToLower(url)
	for domain, name := range platformMap {
		if strings.Contains(urlLower, domain) {
			return name
		}
	}
	return "Unknown"
}

func IsValidURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
