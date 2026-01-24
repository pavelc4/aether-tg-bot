package provider

import (
	"regexp"
	"strings"
)

var (
	urlRegex = regexp.MustCompile(`https?://[^\s]+`)
)

func ExtractURL(text string) string {
	return urlRegex.FindString(text)
}


func IsSupported(url string) bool {
	provider, err := GetProvider(url)
	return err == nil && provider != nil
}


func NormalizeURL(url string) string {
	return strings.TrimSpace(url)
}
