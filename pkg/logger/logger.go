package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	Log = slog.New(handler)
}

func Info(msg string, args ...any) {
	Log.Info(msg, args...)
}

func Error(msg string, args ...any) {
	Log.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	Log.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	Log.Warn(msg, args...)
}
