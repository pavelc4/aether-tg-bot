package middleware

import (
	"runtime/debug"
	"time"

	"github.com/pavelc4/aether-tg-bot/pkg/logger"
)

func Recover(next func()) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Panic recovered", "error", r, "stack", string(debug.Stack()))
			}
		}()
		next()
	}
}

func Logger(name string, next func()) func() {
	return func() {
		start := time.Now()
	
		defer func() {
			duration := time.Since(start)
			if duration > 100 * time.Millisecond {
				logger.Info("Handler completed (slow)", "name", name, "duration", duration)
			} else {
				logger.Debug("Handler completed", "name", name, "duration", duration)
			}
		}()
		
		next()
	}
}

func Chain(f func(), middlewares ...func(func()) func()) func() {
	for i := len(middlewares) - 1; i >= 0; i-- {
		f = middlewares[i](f)
	}
	return f
}
