package core

import (
	"fmt"
	"log"
	"os"
	"time"
)

func NewMetricsCollector(provider string) *MetricsCollector {
	return &MetricsCollector{
		metrics: &DownloadMetrics{
			StartTime: time.Now(),
			Provider:  provider,
		},
	}
}

func (mc *MetricsCollector) RecordFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.FileSize += info.Size()
	mc.metrics.FilesCount++

	return nil
}

func (mc *MetricsCollector) RecordMultipleFiles(filePaths []string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, path := range filePaths {
		info, err := os.Stat(path)
		if err != nil {
			log.Printf("Warning: Failed to stat file %s: %v", path, err)
			continue
		}

		mc.metrics.FileSize += info.Size()
		mc.metrics.FilesCount++
	}

	if mc.metrics.FilesCount == 0 {
		return fmt.Errorf("no valid files recorded")
	}

	return nil
}

func (mc *MetricsCollector) Finish() *DownloadMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics.EndTime = time.Now()
	mc.metrics.LogMetrics()

	return mc.metrics
}

func (mc *MetricsCollector) GetMetrics() *DownloadMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return mc.metrics
}

func FormatDownloadStats(metrics *DownloadMetrics) string {
	if metrics == nil {
		return "No metrics available"
	}

	return fmt.Sprintf(
		"📊 Download Stats:\n"+
			"├─ Provider: %s\n"+
			"├─ Files: %d\n"+
			"├─ Size: %.2f MB\n"+
			"├─ Speed: %.2f MB/s\n"+
			"├─ Duration: %.1fs\n"+
			"└─ Avg Speed: %.2f MB/s",
		metrics.Provider,
		metrics.FilesCount,
		float64(metrics.FileSize)/(1024*1024),
		float64(metrics.FileSize)/(1024*1024)/metrics.Duration.Seconds(),
		metrics.Duration.Seconds(),
		metrics.AvgSpeed,
	)
}
