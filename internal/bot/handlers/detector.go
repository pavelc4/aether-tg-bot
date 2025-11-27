package handlers

import "strings"

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
