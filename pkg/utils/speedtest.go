package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	speedTestDownURL = "https://speed.cloudflare.com/__down?bytes=50000000" // 50MB
	speedTestUpURL   = "https://speed.cloudflare.com/__up"
	cfTraceURL       = "https://speed.cloudflare.com/cdn-cgi/trace"
	speedTestTimeout = 45 * time.Second
)

type SpeedTestResult struct {
	DownloadSpeed   float64       // Mbps
	UploadSpeed     float64       // Mbps
	Latency         time.Duration // ms
	Duration        time.Duration
	BytesDownloaded int64
	ServerLocation  string
	ServerIP        string
	ClientIP        string
	Error           error
}

func RunSpeedTest() *SpeedTestResult {
	result := &SpeedTestResult{}

	ctx, cancel := context.WithTimeout(context.Background(), speedTestTimeout)
	defer cancel()

	// 1. Trace (Location & IP)
	trace, err := fetchTrace(ctx)
	if err == nil {
		result.ServerLocation = fmt.Sprintf("%s (%s)", trace["colo"], trace["loc"])
		result.ClientIP = trace["ip"]
	}

	// 2. Latency
	latency, err := testLatency(ctx)
	if err != nil {
		result.Error = fmt.Errorf("latency test failed: %w", err)
		return result
	}
	result.Latency = latency

	// 3. Download
	downloadSpeed, bytesDownloaded, _, err := testDownloadSpeed(ctx)
	if err != nil {
		result.Error = fmt.Errorf("download test failed: %w", err)
		return result
	}
	result.DownloadSpeed = downloadSpeed
	result.BytesDownloaded = bytesDownloaded

	// 4. Upload
	uploadSpeed, _, _, err := testUploadSpeed(ctx)
	if err != nil {
		// Log error but don't fail entire test?
		// For now let's fail to report full status
		result.Error = fmt.Errorf("upload test failed: %w", err)
		return result
	}
	result.UploadSpeed = uploadSpeed

	return result
}

func fetchTrace(ctx context.Context) (map[string]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", cfTraceURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	data := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			data[parts[0]] = parts[1]
		}
	}
	return data, nil
}

func testLatency(ctx context.Context) (time.Duration, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	// Use trace URL for latency to simulate connection to test server
	req, err := http.NewRequestWithContext(ctx, "HEAD", cfTraceURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return time.Since(start), nil
}

func testDownloadSpeed(ctx context.Context) (float64, int64, time.Duration, error) {
	client := &http.Client{
		Timeout: speedTestTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", speedTestDownURL, nil)
	if err != nil {
		return 0, 0, 0, err
	}

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	downloaded, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}

	duration := time.Since(start)
	seconds := duration.Seconds()
	if seconds <= 0 {
		return 0, 0, 0, fmt.Errorf("invalid test duration")
	}

	speedMbps := (float64(downloaded) * 8) / seconds / 1_000_000

	return speedMbps, downloaded, duration, nil
}

func testUploadSpeed(ctx context.Context) (float64, int64, time.Duration, error) {
	client := &http.Client{
		Timeout: speedTestTimeout,
	}
	size := 20 * 1024 * 1024
	data := make([]byte, size) // 20MB alloc

	req, err := http.NewRequestWithContext(ctx, "POST", speedTestUpURL, bytes.NewReader(data))
	if err != nil {
		return 0, 0, 0, err
	}

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	duration := time.Since(start)
	seconds := duration.Seconds()

	speedMbps := (float64(size) * 8) / seconds / 1_000_000

	return speedMbps, int64(size), duration, nil
}
