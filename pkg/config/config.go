package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	URLs               []string `json:"urls"`
	WorkerCount        int      `json:"worker_count"`
	RateLimitPerSec    int      `json:"rate_limit_per_sec"`
	RequestTimeOutSecs int      `json:"request_timeout_secs"`
	LogLevel           string   `json:"log_level"`
	OutputFile         string   `json:"output_file"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w",err)
	}
	var cfg Config
	if err:=json.Unmarshal(data,&cfg); err!=nil {
		return nil, fmt.Errorf("malformed config: %w",err)
	}
	if len(cfg.URLs) == 0 {
		return nil,fmt.Errorf("no URLs provided in config")
	}
	if cfg.WorkerCount < 5 {
		cfg.WorkerCount=5
		fmt.Println("Worker Count too low, defaulting to 5")
	}
	if cfg.RateLimitPerSec > 2 {
		cfg.RateLimitPerSec = 2
		fmt.Println("Rate limit per sec too high, possible ip block, defaulting to 2 per second.")
	}
	switch cfg.LogLevel{
		case "debug", "warn", "error", "info", "":
		default :
			cfg.LogLevel="info"
			fmt.Println("Unrecognised logging level, defaulting to INFO.")
	}
	if cfg.RequestTimeOutSecs < 0 {
		cfg.RequestTimeOutSecs=5
		fmt.Println("Request time out too small, defaulting to 5 seconds.")
	}
	if cfg.OutputFile == "" {
		cfg.OutputFile="result.json"
		fmt.Println("Output file not specified, defaulting to result.json")
	}

	return &cfg, nil
}