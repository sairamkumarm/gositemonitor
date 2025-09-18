package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/config"
	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"github.com/sairamkumarm/gositemonitor/pkg/scrapper"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.json", "Load a configuration for the site monitor")
	runtimeTimout := flag.Int("runtime", -100, "Monitor runtime in seconds")
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


	var finish context.Context
	var cancel context.CancelFunc
	if *runtimeTimout != -100 {
		finish, cancel = context.WithTimeout(context.Background(), time.Duration(*runtimeTimout)*time.Second)
		defer cancel()
	} else {
		finish, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	sigCh := make(chan os.Signal,1)
	signal.Notify(sigCh,syscall.SIGINT,syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("Interrupt acknowledged")
		cancel()
	}()


	jobs := make(chan string, len(config.URLs))
	results := make(chan scrapper.ScrapeResult)

	//create permit channel that releases the ratelimit amount of tokens every second, so the workers can pick them up and work
	permits := make(chan struct{}, config.RateLimitPerSec)
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(config.RateLimitPerSec))
		defer ticker.Stop()
	mainloop:
		for {
			<-ticker.C
			select {
			case <-finish.Done():
				break mainloop
			case permits <- struct{}{}:
			}
		}
	}()

	//job refiller
	go func() {
		ticker := time.NewTicker(config.RequestIntervalDuration())
		defer ticker.Stop()
	mainloop:
		for {
			for _, url := range config.URLs {
				for i := 0; i < config.RateLimitPerSec; i++ {
					select {
					case <-finish.Done():
						break mainloop
					case jobs <- url:
						//enqueue
					}
				}
			}
			<-ticker.C

		}
	}()

	timeout := time.Duration(config.RequestTimeOutSecs) * time.Second
	//resuse a shared httpclient in all the workers, common transport settings are configured here
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableCompression:    false,
		},
	}
	//spawn workers, they wait internally for jobs and permits from their channels
	for i := 0; i < config.WorkerCount; i++ {
		go scrapper.Worker(i, jobs, results, permits, timeout, client, finish)
	}

	//read results channel and log outputs
	go func() {
		for res := range results {
			log.Info("scrape done",
				zap.String("url", res.URL),
				zap.Int("status", res.Status),
				zap.Int64("latency_ms", res.ResponseMS),
				zap.String("err", res.Error),
				zap.Int("Worker", res.WorkerID),
			)
		}
	}()

		
		<-finish.Done()
		fmt.Println("Shutting down")
	}
