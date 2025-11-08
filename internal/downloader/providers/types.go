package providers

import (
	"context"
)

type Provider interface {
	Name() string
	CanHandle(url string) bool
	Download(ctx context.Context, url string, audioOnly bool) ([]string, error)
}

type DownloadOptions struct {
	URL        string
	AudioOnly  bool
	UseCookies bool
}
