package logger

import (
	"fmt"
	"log/slog"
	"os"
)

var logger *slog.Logger

func Init() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger = slog.New(handler)
}

func Info(format string, args ...interface{}) {
	logger.Info(fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	logger.Error(fmt.Sprintf(format, args...))
}

func Warn(format string, args ...interface{}) {
	logger.Warn(fmt.Sprintf(format, args...))
}
