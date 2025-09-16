package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/config"
	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"github.com/sairamkumarm/gositemonitor/pkg/scrapper"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.json", "Load a configuration for the site monitor")
	flag.Parse()

	// config, err := config.Load()
	config, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(config.LogLevel)
	defer log.Sync()

	log.Info("Starting GoSiteMonitor",
		zap.Int("urls_count", len(config.URLs)),
		zap.String("output_file", config.OutputFile),
		zap.Int("worker_count", config.WorkerCount),
		zap.Int("rate_limit_per_sec", config.RateLimitPerSec),
		zap.Int("request_timeout", config.RequestTimeOutSecs))

	b, _ := json.MarshalIndent(config, "", " ")
	log.Info("loaded config", zap.String("config", string(b)))
	fmt.Println("Ready to commence operations.")

	jobs := make(chan string, len(config.URLs))
	results := make(chan scrapper.ScrapeResult)

	//create permit channel that releases the ratelimit amount of tokens every second, so the workers can pick them up and work
	permits := make(chan struct{}, config.RateLimitPerSec)
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(config.RateLimitPerSec))
		defer ticker.Stop()
		for range ticker.C {
			permits <- struct{}{}
		}
	}()

	// for _, url := range config.URLs {
	// 	jobs <- url
	// }
	// refill jobs channel with jobs every 3 seconds
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			for _,url:= range config.URLs{
				for i:=0 ; i<config.RateLimitPerSec; i++{
					jobs<-url
				}
			}
			<-ticker.C
		}
	}()

	//spawn workers, internally they wait for jobs and permits
	for i := 0; i < config.WorkerCount; i++ {
		go scrapper.Worker(i, jobs, results, permits)
	}

	//read results channel and log outputs
	go func() {
		for res := range results {
			log.Info("scrape done",
				zap.String("url", res.URL),
				zap.Int("status", res.Status),
				zap.Int64("latency_ms", res.ResponseMS),
				zap.String("err", res.Error),
				zap.Int("Worker",res.WorkerID),
			)
		}
	}()


	select {}
}
