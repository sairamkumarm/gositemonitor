package logger

import (
	"strings"
	"github.com/sairamkumarm/gositemonitor/pkg/scrapper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func New(logLevel string) {

	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder //uses time format 2025-09-21T18:32:00Z
	switch strings.ToLower(logLevel) {
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
	Log = logger
}

func ResultLogger(res scrapper.ScrapeResult){
	switch {
			case res.Status == -1:
				Log.Error("Ping Failed",
					zap.String("URL", res.URL),
					zap.Time("TimeStampUTC", res.TimestampUTC),
					zap.String("Error", res.Error),
					zap.Int("WorkerID", res.WorkerID))
			case res.Status >= 400:
				Log.Warn("Non-2XX Status",
					zap.String("URL", res.URL),
					zap.Time("TimeStampUTC", res.TimestampUTC),
					zap.Int("Status", res.Status),
					zap.Int("Latency", int(res.ResponseMS)),
					zap.Int("WorkerID", res.WorkerID))
			default:
				Log.Info("Ping Success",
					zap.String("URL", res.URL),
					zap.Time("TimeStampUTC", res.TimestampUTC),
					zap.Int("Status", res.Status),
					zap.Int("Latency", int(res.ResponseMS)),
					zap.Int("WorkerID", res.WorkerID))
			}
}
