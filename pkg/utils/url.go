package utils

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func ResolveFinalURL(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	resp, err := GetDownloadClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve URL failed: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if finalURL != url {
		log.Printf("URL resolved: %s -> %s", url, finalURL)
	}

	return finalURL, nil
}
