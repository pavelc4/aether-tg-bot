package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	tikTokAPIURL  = "https://www.tikwm.com/api/"
	tikTokTimeout = 30 * time.Second
)

type TikTokProvider struct {
	client *http.Client
}

func NewTikTok() *TikTokProvider {
	return &TikTokProvider{
		client: &http.Client{
			Timeout: tikTokTimeout,
		},
	}
}

func (tp *TikTokProvider) Name() string {
	return "TikTok"
}

func (tp *TikTokProvider) Supports(url string) bool {
	return strings.Contains(url, "tiktok.com") || strings.Contains(url, "vt.tiktok.com")
}

func (tp *TikTokProvider) GetVideoInfo(ctx context.Context, url string, opts Options) ([]VideoInfo, error) {
	resp, err := tp.fetchData(ctx, url)
	if err != nil {
		return nil, err
	}

	if opts.AudioOnly && resp.Data.Music != "" {
		musicURL := resp.Data.Music
		if !strings.HasPrefix(musicURL, "http") {
			musicURL = "https://tikwm.com" + musicURL
		}
		return []VideoInfo{{
			URL:      musicURL,
			FileName: fmt.Sprintf("tiktok_audio_%s.mp3", resp.Data.ID),
			Title:    resp.Data.Title,
			FileSize: 0,
			MimeType: "audio/mpeg",
			Duration: resp.Data.Duration,
		}}, nil
	}

	videoURL := resp.Data.Play
	if videoURL == "" {
		return nil, fmt.Errorf("video URL not found in response")
	}
	if !strings.HasPrefix(videoURL, "http") {
		videoURL = "https://tikwm.com" + videoURL
	}

	return []VideoInfo{{
		URL:      videoURL,
		FileName: fmt.Sprintf("tiktok_%s.mp4", resp.Data.ID),
		Title:    resp.Data.Title,
		FileSize: int64(resp.Data.Size), // TikWM provides size
		MimeType: "video/mp4",
		Duration: resp.Data.Duration,
	}}, nil
}

type tikWMResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data tikWMData  `json:"data"`
}

type tikWMData struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Play     string `json:"play"`    // Video URL
	Music    string `json:"music"`   // Audio URL
	Size     int    `json:"size"`
	Duration int    `json:"duration"`
}

func (tp *TikTokProvider) fetchData(ctx context.Context, tiktokURL string) (*tikWMResponse, error) {
	payload := map[string]string{"url": tiktokURL}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tikTokAPIURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result tikWMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API error: %s", result.Msg)
	}

	return &result, nil
}
