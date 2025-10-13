package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/pavelc4/aether-tg-bot/bot"
	"github.com/pavelc4/aether-tg-bot/config"
)

const (
	shutdownTimeout = 30 * time.Second
	maxRetries      = 3
	retryDelay      = 5 * time.Second
)

func main() {
	// Setup panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal panic: %v", r)
			// Print stack trace
			buf := make([]byte, 4096)
			n := runtime.Stack(buf, false)
			log.Printf("Stack trace:\n%s", buf[:n])
			os.Exit(1)
		}
	}()

	// Setup structured logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Aether Telegram Bot...")

	// Print version info
	printVersionInfo()

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Get bot token
	token := config.GetBotToken()
	if token == "" {
		log.Fatal("BOT_TOKEN is not set")
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	// Start bot dengan retry mechanism
	errChan := make(chan error, 1)
	go func() {
		if err := startBotWithRetry(ctx, token, maxRetries); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal atau error
	select {
	case sig := <-sigChan:
		handleShutdown(ctx, cancel, sig)

	case err := <-errChan:
		log.Fatalf("Bot failed after %d retries: %v", maxRetries, err)
	}
}

// startBotWithRetry starts bot dengan retry mechanism
func startBotWithRetry(ctx context.Context, token string, maxRetries int) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		log.Printf("Starting bot (attempt %d/%d)...", attempt, maxRetries)

		if err := bot.StartBot(token); err != nil {
			lastErr = err
			log.Printf("Bot failed: %v. Retrying in %v...", err, retryDelay)

			if attempt < maxRetries {
				time.Sleep(retryDelay)
				continue
			}

			return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
		}

		// Bot stopped normally
		return nil
	}

	return lastErr
}

// handleShutdown performs graceful shutdown
func handleShutdown(ctx context.Context, cancel context.CancelFunc, sig os.Signal) {
	log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)

	// Cancel context untuk stop bot
	cancel()

	// Wait for cleanup dengan timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	// Perform cleanup
	performCleanup(shutdownCtx)

	log.Println("Shutdown complete. Goodbye!")
}

// performCleanup handles graceful shutdown tasks
func performCleanup(ctx context.Context) {
	log.Println("Performing cleanup...")

	done := make(chan struct{})

	go func() {
		// Cleanup tasks dengan context awareness
		cleanupTempFiles(ctx)
		flushLogs()

		close(done)
	}()

	// Wait for cleanup atau timeout
	select {
	case <-done:
		log.Println("Cleanup completed successfully")
	case <-ctx.Done():
		log.Println("Cleanup timeout exceeded, forcing shutdown")
	}
}

// cleanupTempFiles removes temporary files dengan context
func cleanupTempFiles(ctx context.Context) {
	log.Println("Cleaning up temporary files...")

	// Get temp directory patterns
	patterns := []string{
		"aether-ytdlp-*",
		"aether-scrape-*",
		"aether-image-*",
		"aether-tiktok-audio-*",
	}

	tempDir := os.TempDir()
	filesCleaned := 0

	for _, pattern := range patterns {
		// Check context cancellation sebelum setiap pattern
		select {
		case <-ctx.Done():
			log.Println("Cleanup cancelled due to timeout")
			return
		default:
		}

		fullPattern := filepath.Join(tempDir, pattern)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			log.Printf("Error finding temp files with pattern %s: %v", pattern, err)
			continue
		}

		for _, path := range matches {
			// Check context cancellation sebelum setiap file operation
			select {
			case <-ctx.Done():
				log.Printf("Cleanup cancelled during file operations (%d files cleaned)", filesCleaned)
				return
			default:
			}

			if err := os.RemoveAll(path); err != nil {
				log.Printf("Failed to remove %s: %v", path, err)
			} else {
				filesCleaned++
				log.Printf("Cleaned up: %s", filepath.Base(path))
			}
		}
	}

	log.Printf("Temp files cleanup completed. %d files/directories cleaned.", filesCleaned)
}

// flushLogs ensures all logs are written
func flushLogs() {
	log.Println("Flushing logs...")
	// Force sync stdout/stderr
	os.Stdout.Sync()
	os.Stderr.Sync()
}

// printVersionInfo prints version and runtime info
func printVersionInfo() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("CPU Cores: %d", runtime.NumCPU())
	log.Printf("Goroutines: %d", runtime.NumGoroutine())

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("Memory Allocated: %.2f MB", float64(m.Alloc)/1024/1024)
}
