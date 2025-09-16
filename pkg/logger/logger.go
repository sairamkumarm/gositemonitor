package logger

import (
	"strings"

	"go.uber.org/zap"
)

func New(level string) *zap.Logger {

	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"

	switch strings.ToLower(level) {
	case "info":
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "debug":
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "error":
		cfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, _ := cfg.Build()
	return logger
}
