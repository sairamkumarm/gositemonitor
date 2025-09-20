package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	URLs               []string `json:"urls"`
	WorkerCount        int      `json:"worker_count"`
	RateLimitPerSec    int      `json:"rate_limit_per_sec"`
	RequestTimeOutSecs int      `json:"request_timeout_secs"`
	LogLevel           string   `json:"log_level"`
	OutputDir         string   `json:"output_dir"`
	RequestInterval    int      `json:"request_interval"`
}

func Load(path string) (*Config, error) {

	const (
		minWorkers      = 1
		defaultWorkers  = 5
		maxWorkers      = 500
		minTimeoutSecs  = 1
		defaultTimeout  = 5
		minIntervalSecs = 5
		defaultInterval = 10
		minRatePerSec   = 1
		maxRatePerSec   = 10
	)


	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("malformed config: %w", err)
	}


	if len(cfg.URLs) == 0 {
		return nil, fmt.Errorf("no URLs provided in config")
	}

	cleanedURLs := make([]string, 0,len(cfg.URLs))

	for i, u := range cfg.URLs {
		u = strings.TrimSpace(u)
		if u == "" {
			fmt.Printf("Skipping empty URL at index %d\n",i)
			continue
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme=="" || parsed.Host == ""{
			return nil, fmt.Errorf("invalid URL at index %d: %q", i, u)
		}
		if parsed.Scheme!="http" && parsed.Scheme!="https" {
			return nil, fmt.Errorf("unsupported URL scheme at index %d: %q",i,u)
		}
		cleanedURLs = append(cleanedURLs, parsed.String())
	}
	if len(cleanedURLs)==0 {
		return nil, fmt.Errorf("no Valid URLs to monitor")
	}
	cfg.URLs = cleanedURLs

		// Worker count
	if cfg.WorkerCount < minWorkers {
		fmt.Printf("WorkerCount too low (%d), defaulting to %d\n", cfg.WorkerCount, defaultWorkers)
		cfg.WorkerCount = defaultWorkers
	}
	if cfg.WorkerCount > maxWorkers {
		fmt.Printf("WorkerCount too high (%d), capping to %d\n", cfg.WorkerCount, maxWorkers)
		cfg.WorkerCount = maxWorkers
	}

	// Rate limit per second (global)
	if cfg.RateLimitPerSec < minRatePerSec {
		fmt.Printf("RateLimitPerSec too low (%d), defaulting to %d\n", cfg.RateLimitPerSec, minRatePerSec)
		cfg.RateLimitPerSec = minRatePerSec
	}
	if cfg.RateLimitPerSec > maxRatePerSec {
		fmt.Printf("RateLimitPerSec too high (%d), capping to %d to avoid accidental DOS\n", cfg.RateLimitPerSec, maxRatePerSec)
		cfg.RateLimitPerSec = maxRatePerSec
	}

	// Request timeout
	if cfg.RequestTimeOutSecs < minTimeoutSecs {
		fmt.Printf("RequestTimeOutSecs too small (%d), defaulting to %d seconds\n", cfg.RequestTimeOutSecs, defaultTimeout)
		cfg.RequestTimeOutSecs = defaultTimeout
	}

	// Request interval between bursts
	if cfg.RequestInterval < minIntervalSecs {
		fmt.Printf("RequestInterval out too short (%d), defaulting to %d seconds\n", cfg.RequestInterval, defaultInterval)
		cfg.RequestInterval = defaultInterval
	}

	// Safety: avoid accidental overlaps by default
	if cfg.RequestInterval <= cfg.RequestTimeOutSecs {
		// Add one second buffer so most requests from previous burst can finish
		newInterval := cfg.RequestTimeOutSecs + 1
		fmt.Printf("RequestInterval (%d) <= RequestTimeOutSecs (%d), bumping RequestInterval to %d to avoid burst overlap\n",
			cfg.RequestInterval, cfg.RequestTimeOutSecs, newInterval)
		// Only increase, never decrease here
		cfg.RequestInterval = newInterval
	}

	if strings.TrimSpace(cfg.OutputDir) == "" {
		cfg.OutputDir = "gsm_logs"
		fmt.Println("Output Directory not specified, defaulting to gsm_logs")
	}

	switch strings.ToLower(cfg.LogLevel) {
	case "debug", "warn", "error", "info", "":
	default:
		cfg.LogLevel = "info"
		fmt.Println("Unrecognized log level, defaulting to info")
	}

	return &cfg, nil
}
func (c *Config) RequestIntervalDuration() time.Duration {
	return time.Duration(c.RequestInterval) * time.Second
}
