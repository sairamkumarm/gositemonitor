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
	OutputDir          string   `json:"output_dir"`
	RequestInterval    int      `json:"request_interval"`
}
var ProdConfig Config = Config{}

func Load(path string) (error) {
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
		return fmt.Errorf("cannot read config: %w", err)
	}
	if err := json.Unmarshal(data, &ProdConfig); err != nil {
		return fmt.Errorf("malformed config: %w", err)
	}

	if len(ProdConfig.URLs) == 0 {
		return fmt.Errorf("no URLs provided in config")
	}

	urlmap := make(map[string]struct{}) //handling duplicate urls
	cleanedURLs := make([]string, 0, len(ProdConfig.URLs))

	for i, u := range ProdConfig.URLs {
		u = strings.TrimSpace(u)
		if u == "" {
			fmt.Printf("Skipping empty URL at index %d\n", i)
			continue
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid URL at index %d: %q", i, u)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("unsupported URL scheme at index %d: %q", i, u)
		}
		_, duplicate := urlmap[parsed.String()] //duplicate handling
		if duplicate {
			fmt.Printf("Skipping duplicate URL %s\n", parsed.String())
		} else {
			urlmap[parsed.String()] = struct{}{}
			cleanedURLs = append(cleanedURLs, parsed.String())
		}
	}
	if len(cleanedURLs) == 0 {
		return fmt.Errorf("no Valid URLs to monitor")
	}
	ProdConfig.URLs = cleanedURLs

	// Worker count
	if ProdConfig.WorkerCount < minWorkers {
		fmt.Printf("WorkerCount too low (%d), defaulting to %d\n", ProdConfig.WorkerCount, defaultWorkers)
		ProdConfig.WorkerCount = defaultWorkers
	}
	if ProdConfig.WorkerCount > maxWorkers {
		fmt.Printf("WorkerCount too high (%d), capping to %d\n", ProdConfig.WorkerCount, maxWorkers)
		ProdConfig.WorkerCount = maxWorkers
	}

	// Rate limit per second (global)
	if ProdConfig.RateLimitPerSec < minRatePerSec {
		fmt.Printf("RateLimitPerSec too low (%d), defaulting to %d\n", ProdConfig.RateLimitPerSec, minRatePerSec)
		ProdConfig.RateLimitPerSec = minRatePerSec
	}
	if ProdConfig.RateLimitPerSec > maxRatePerSec {
		fmt.Printf("RateLimitPerSec too high (%d), capping to %d to avoid accidental DOS\n", ProdConfig.RateLimitPerSec, maxRatePerSec)
		ProdConfig.RateLimitPerSec = maxRatePerSec
	}

	// Request timeout
	if ProdConfig.RequestTimeOutSecs < minTimeoutSecs {
		fmt.Printf("RequestTimeOutSecs too small (%d), defaulting to %d seconds\n", ProdConfig.RequestTimeOutSecs, defaultTimeout)
		ProdConfig.RequestTimeOutSecs = defaultTimeout
	}

	// Request interval between bursts
	if ProdConfig.RequestInterval < minIntervalSecs {
		fmt.Printf("RequestInterval out too short (%d), defaulting to %d seconds\n", ProdConfig.RequestInterval, defaultInterval)
		ProdConfig.RequestInterval = defaultInterval
	}

	// Safety: avoid accidental overlaps by default
	if ProdConfig.RequestInterval <= ProdConfig.RequestTimeOutSecs {
		// Add one second buffer so most requests from previous burst can finish
		newInterval := ProdConfig.RequestTimeOutSecs + 1
		fmt.Printf("RequestInterval (%d) <= RequestTimeOutSecs (%d), bumping RequestInterval to %d to avoid burst overlap\n",
			ProdConfig.RequestInterval, ProdConfig.RequestTimeOutSecs, newInterval)
		// Only increase, never decrease here
		ProdConfig.RequestInterval = newInterval
	}

	if strings.TrimSpace(ProdConfig.OutputDir) == "" {
		ProdConfig.OutputDir = "gsm_logs"
		fmt.Println("Output Directory not specified, defaulting to gsm_logs")
	}

	switch strings.ToLower(ProdConfig.LogLevel) {
	case "debug", "warn", "error", "info", "":
	default:
		ProdConfig.LogLevel = "info"
		fmt.Println("Unrecognized log level, defaulting to info")
	}

	return nil
}
func (c *Config) GetRequestIntervalDuration() time.Duration {
	return time.Duration(c.RequestInterval) * time.Second
}
