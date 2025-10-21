package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	speedTestURL     = "https://speed.cloudflare.com/__down?bytes=10000000" // 10MB
	speedTestTimeout = 30 * time.Second
	latencyTestURL   = "https://www.google.com"
)

// SpeedTestResult contains speed test results
type SpeedTestResult struct {
	DownloadSpeed   float64       // Mbps
	UploadSpeed     float64       // Mbps (not implemented yet)
	Latency         time.Duration // ms
	Duration        time.Duration
	BytesDownloaded int64
	Error           error
}

// RunSpeedTest performs network speed test
func RunSpeedTest() *SpeedTestResult {
	result := &SpeedTestResult{}

	ctx, cancel := context.WithTimeout(context.Background(), speedTestTimeout)
	defer cancel()

	// Test latency
	latency, err := testLatency(ctx)
	if err != nil {
		result.Error = fmt.Errorf("latency test failed: %w", err)
		return result
	}
	result.Latency = latency

	// Test download speed
	downloadSpeed, bytesDownloaded, duration, err := testDownloadSpeed(ctx)
	if err != nil {
		result.Error = fmt.Errorf("download test failed: %w", err)
		return result
	}

	result.DownloadSpeed = downloadSpeed
	result.BytesDownloaded = bytesDownloaded
	result.Duration = duration

	return result
}

// testLatency tests network latency
func testLatency(ctx context.Context) (time.Duration, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", latencyTestURL, nil)
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

// testDownloadSpeed tests download speed
func testDownloadSpeed(ctx context.Context) (float64, int64, time.Duration, error) {
	client := &http.Client{
		Timeout: speedTestTimeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", speedTestURL, nil)
	if err != nil {
		return 0, 0, 0, err
	}

	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	// Download and discard data
	downloaded, err := io.Copy(io.Discard, resp.Body)
	if err != nil {
		return 0, 0, 0, err
	}

	duration := time.Since(start)

	// Calculate speed in Mbps
	seconds := duration.Seconds()
	if seconds <= 0 {
		return 0, 0, 0, fmt.Errorf("invalid test duration")
	}

	// Bytes -> Megabits per second
	speedMbps := (float64(downloaded) * 8) / seconds / 1_000_000

	return speedMbps, downloaded, duration, nil
}
