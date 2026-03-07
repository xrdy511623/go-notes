package performance

import (
	"context"
	"io"
	"log"
	"log/slog"
)

// StdLogger 标准库log的封装
type StdLogger struct {
	logger *log.Logger
}

func NewStdLogger() *StdLogger {
	return &StdLogger{logger: log.New(io.Discard, "", log.LstdFlags)}
}

func (l *StdLogger) Info(msg string, args ...any) {
	l.logger.Printf(msg, args...)
}

// SlogLogger slog的Text/JSON logger
func NewSlogText() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func NewSlogJSON() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func NewSlogJSONWithLevel(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: level}))
}

// SlogAttrsRecord 用于对比交替key-value和强类型Attr的性能
func LogWithKeyValue(logger *slog.Logger) {
	logger.Info("request completed",
		"method", "GET",
		"path", "/api/users",
		"status", 200,
		"bytes", 1024,
	)
}

func LogWithAttrs(logger *slog.Logger) {
	logger.LogAttrs(context.Background(), slog.LevelInfo, "request completed",
		slog.String("method", "GET"),
		slog.String("path", "/api/users"),
		slog.Int("status", 200),
		slog.Int("bytes", 1024),
	)
}
