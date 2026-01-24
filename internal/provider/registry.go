package provider

import (
	"context"
	"fmt"
	"net/url"
	"sync"
)

var (
	registry = make([]Provider, 0)
	mu       sync.RWMutex
)


func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry = append(registry, p)
}


func GetProvider(rawURL string) (Provider, error) {

	_, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, p := range registry {
		if p.Supports(rawURL) {
			return p, nil
		}
	}

	return nil, fmt.Errorf("no provider found for this URL")
}


func Resolve(ctx context.Context, url string) (*VideoInfo, string, error) {
	p, err := GetProvider(url)
	if err != nil {
		return nil, "", err
	}

	info, err := p.GetVideoInfo(ctx, url)
	if err != nil {
		return nil, p.Name(), err
	}

	return info, p.Name(), nil
}
