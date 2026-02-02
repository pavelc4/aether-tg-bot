package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/pavelc4/aether-tg-bot/config"
)

const cobaltTimeout = 30 * time.Second

type CobaltProvider struct {
	client *http.Client
}

func NewCobalt() *CobaltProvider {
	return &CobaltProvider{
		client: &http.Client{
			Timeout: cobaltTimeout,
		},
	}
}

func (cp *CobaltProvider) Name() string {
	return "Cobalt"
}

func (cp *CobaltProvider) Supports(url string) bool {
	if strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be") {
		return false
	}
	supportedDomains := []string{
		"instagram.com", 
		"instagr.am",
		"twitter.com", 
		"x.com",
		"tiktok.com", 
		"vm.tiktok.com",
		"vt.tiktok.com",
		"threads.net",
		"soundcloud.com",
		"spotify.com",
		"reddit.com", 
		"redd.it",
		"twitch.tv",
		"facebook.com", 
		"fb.watch",
		"vimeo.com",
		"pinterest.com", 
		"pin.it",
		"streamable.com",
		"bilibili.com",
		"dailymotion.com",
		"dai.ly",
		"vk.com",
		"tumblr.com",
	}

	for _, d := range supportedDomains {
		if strings.Contains(url, d) {
			return true
		}
	}
	
	return false
}

func (cp *CobaltProvider) GetVideoInfo(ctx context.Context, url string, opts Options) ([]VideoInfo, error) {
	apiResp, err := cp.requestAPI(ctx, url, opts)
	if err != nil {
		return nil, err
	}

	return cp.parseResponse(apiResp)
}

type cobaltAPIResponse struct {
	Status   string        `json:"status"`
	URL      string        `json:"url"`
	Filename string        `json:"filename"`
	Picker   []cobaltItem  `json:"picker"`
	Error    cobaltError   `json:"error"`
}

type cobaltItem struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	Filename string `json:"filename"`
}

type cobaltError struct {
	Code    string `json:"code"`
	Context string `json:"context"`
}

func (cp *CobaltProvider) requestAPI(ctx context.Context, mediaURL string, opts Options) (*cobaltAPIResponse, error) {
	requestBody := map[string]interface{}{
		"url":          mediaURL,
		"downloadMode": "auto",
		"videoQuality": "max",
	}
	
	if opts.AudioOnly {
		requestBody["downloadMode"] = "audio"
		requestBody["isAudioOnly"] = true
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	apiURL := config.GetCobaltAPI()
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if apiKey := config.GetCobaltAPIKey(); apiKey != "" {
		req.Header.Set("Authorization", "Api-Key "+apiKey)
	}

	resp, err := cp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cobalt request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cobalt returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var cobaltResponse cobaltAPIResponse
	if err := json.Unmarshal(bodyBytes, &cobaltResponse); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	return &cobaltResponse, nil
}

func (cp *CobaltProvider) parseResponse(resp *cobaltAPIResponse) ([]VideoInfo, error) {
	switch resp.Status {
	case "tunnel", "redirect":
		if resp.URL == "" {
			return nil, fmt.Errorf("empty URL in cobalt response")
		}
		return []VideoInfo{{
			URL:      resp.URL,
			FileName: resp.Filename,
			Title:    resp.Filename,
			MimeType: guessMimeType(resp.Filename),
		}}, nil

	case "picker":
		if len(resp.Picker) == 0 {
			return nil, fmt.Errorf("empty picker in cobalt response")
		}
		
		var results []VideoInfo
		for _, item := range resp.Picker {
			if item.URL == "" {
				continue
			}
			filename := item.Filename
			if filename == "" {
				filename = resp.Filename
			}
			if filename == "" {
				ext := ".jpg"
				if item.Type == "video" {
					ext = ".mp4"
				}
				filename = fmt.Sprintf("cobalt_%d_%d%s", time.Now().Unix(), len(results), ext)
			}
			
			results = append(results, VideoInfo{
				URL:      item.URL,
				FileName: filename,
				Title:    filename,
				MimeType: guessMimeType(filename),
			})
		}
		
		if len(results) == 0 {
			return nil, fmt.Errorf("no valid items found in picker")
		}

		return results, nil

	case "error":
		return nil, fmt.Errorf("cobalt API error: %s (%s)", resp.Error.Code, resp.Error.Context)

	default:
		return nil, fmt.Errorf("unknown cobalt status: %s", resp.Status)
	}
}

var mimeTypes = map[string]string{
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".mkv":  "video/x-matroska",
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".ogg":  "audio/ogg",
	".wav":  "audio/wav",
	".opus": "audio/opus",
	".flac": "audio/flac",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
}

func guessMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if idx := strings.Index(ext, "?"); idx != -1 {
		ext = ext[:idx]
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
