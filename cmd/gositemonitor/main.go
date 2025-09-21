package main

import (
	"context"
	"sync"
	// "encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/aggregator"
	"github.com/sairamkumarm/gositemonitor/pkg/analyser"
	"github.com/sairamkumarm/gositemonitor/pkg/config"
	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"github.com/sairamkumarm/gositemonitor/pkg/notification"
	"github.com/sairamkumarm/gositemonitor/pkg/scheduler"
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

	//create reusable logger
	logger.New(config.LogLevel)
	defer logger.Log.Sync()

	logger.Log.Info("Starting GoSiteMonitor",
		zap.Int("urls_count", len(config.URLs)),
		zap.String("output_dir", config.OutputDir),
		zap.Int("worker_count", config.WorkerCount),
		zap.Int("rate_limit_per_sec", config.RateLimitPerSec),
		zap.Int("request_timeout", config.RequestTimeOutSecs),
		zap.Int("request_interval", config.RequestInterval))

	// b, _ := json.MarshalIndent(config, "", " ")
	// Log.Info("loaded config", zap.String("config", string(b)))

	//create resuable context for total graceful shutdown
	var finish context.Context
	var cancel context.CancelFunc
	if *runtimeTimout != -100 {
		finish, cancel = context.WithTimeout(context.Background(), time.Duration(*runtimeTimout)*time.Second)
		defer cancel()
	} else {
		finish, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	//create signal channel to intercept system calls to kill, resulting in graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	// go routine waiting for an syscall to kill program
	go func() {
		<-sigCh
		fmt.Println("Interrupt acknowledged")
		cancel()
	}()

	//create a waitgroup so all go routines can send shutdown messages
	wg := sync.WaitGroup{}

	//create output dir, if present use that
	err = os.MkdirAll(config.OutputDir, 0755)
	if err != nil {
		logger.Log.Error("Error making output dir: ", zap.Error(err))
	} else {
		logger.Log.Info("Ping results stored in " + config.OutputDir)
	}

	//Initialize stats map
	analyser.FillInitialUrls(config.URLs)

	fmt.Println("Ready to commence operations.")

	jobs := make(chan string, len(config.URLs))
	results := make(chan scrapper.ScrapeResult,100)
	permits := make(chan struct{}, config.RateLimitPerSec)

	//create permit channel that releases the ratelimit amount of tokens every second, so the workers can pick them up and work
	wg.Add(1)//wait for permit handler
	go scheduler.PermitHandler(permits, config.RateLimitPerSec, finish, &wg)

	//job refiller to fill jobs channel periodically with urls to ping
	wg.Add(1)//wait for job refiller
	go scheduler.JobHandler(jobs, config.URLs, config.RateLimitPerSec, config.RequestIntervalDuration(), finish, &wg)

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
		wg.Add(1)//wait for worker
		go scrapper.Worker(i, jobs, results, permits, timeout, client, finish, &wg)

	}

	//read results channel and log outputs
	wg.Add(1)//wait for aggregator
	go aggregator.Aggregate(results, config.OutputDir, finish, cancel, &wg)

	//initiate the notification handler
	wg.Add(1)//wait for notification handler
	go notification.NotificationHandler(config.OutputDir,finish, cancel, &wg)


	<-finish.Done()
	wg.Wait()
	fmt.Println("Shutting down")
}
