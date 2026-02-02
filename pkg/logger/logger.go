package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Cyan    = "\033[36m"
	Gray    = "\033[90m"
)

var levelStyles = map[slog.Level]struct {
	color string
	label string
}{
	slog.LevelDebug: {Gray, "DEBUG"},
	slog.LevelInfo:  {Green, "INFO "},
	slog.LevelWarn:  {Yellow, "WARN "},
	slog.LevelError: {Red, "ERROR"},
}

var Log *slog.Logger

type PrettyHandler struct {
	out        io.Writer
	level      slog.Level
	mu         *sync.Mutex
	timeFormat string
}

func NewPrettyHandler(out io.Writer, level slog.Level, timeFormat string) *PrettyHandler {
	if timeFormat == "" {
		timeFormat = "2006-01-02 15:04:05"
	}
	return &PrettyHandler{
		out:        out,
		level:      level,
		mu:         &sync.Mutex{},
		timeFormat: timeFormat,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	timeStr := r.Time.Format(h.timeFormat)

	style := levelStyles[r.Level]
	if style.label == "" {
		style = levelStyles[slog.LevelInfo]
	}

	line := fmt.Sprintf("%s[AETHER]%s %s %s|%s %s%s%s %s|%s %s",
		Cyan, Reset,
		timeStr,
		Gray, Reset,
		style.color, style.label, Reset,
		Gray, Reset,
		r.Message,
	)

	r.Attrs(func(a slog.Attr) bool {
		line += fmt.Sprintf(" %s%s%s=%v", Cyan, a.Key, Reset, a.Value.Any())
		return true
	})

	line += "\n"
	_, err := h.out.Write([]byte(line))
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return h
}

func init() {
	handler := NewPrettyHandler(os.Stdout, slog.LevelInfo, "")
	Log = slog.New(handler)
	slog.SetDefault(Log)
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

func InfoWithDuration(msg string, start time.Time, args ...any) {
	args = append(args, "duration", time.Since(start).Round(time.Millisecond))
	Log.Info(msg, args...)
}

func ErrorWithDuration(msg string, start time.Time, args ...any) {
	args = append(args, "duration", time.Since(start).Round(time.Millisecond))
	Log.Error(msg, args...)
}
