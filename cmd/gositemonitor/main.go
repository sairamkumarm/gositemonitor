package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/aggregator"
	"github.com/sairamkumarm/gositemonitor/pkg/analyser"
	"github.com/sairamkumarm/gositemonitor/pkg/config"
	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"github.com/sairamkumarm/gositemonitor/pkg/notification"
	"github.com/sairamkumarm/gositemonitor/pkg/pinger"
	"github.com/sairamkumarm/gositemonitor/pkg/scheduler"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.json", "Load a configuration for the site monitor")
	runtimeTimout := flag.Int("runtime", -100, "Monitor runtime in seconds")
	flag.Parse()

	// loads values into a global config struct
	err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	//create reusable logger
	logger.New(config.ProdConfig.LogLevel)
	defer logger.Log.Sync()

	logger.Log.Info("Starting GoSiteMonitor",
		zap.Any("config", config.ProdConfig))

	// b, _ := json.MarshalIndent(config.ProdConfig, "", " ")
	// logger.Log.Info("loaded config", zap.String("config", string(b)))

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
	err = os.MkdirAll(config.ProdConfig.OutputDir, 0755)
	if err != nil {
		logger.Log.Error("Error making output dir: ", zap.Error(err))
	} else {
		logger.Log.Info("Ping results stored in " + config.ProdConfig.OutputDir)
	}

	//Initialize stats map
	analyser.FillInitialUrls(config.ProdConfig.URLs)

	fmt.Println("Ready to commence operations.")

	jobs := make(chan string, len(config.ProdConfig.URLs))
	results := make(chan pinger.PingResult, 100)
	permits := make(chan struct{}, config.ProdConfig.RateLimitPerSec)

	//create permit channel that releases the ratelimit amount of tokens every second, so the workers can pick them up and work
	wg.Add(1) //wait for permit handler
	go scheduler.PermitHandler(permits, config.ProdConfig.RateLimitPerSec, finish, &wg)

	//job refiller to fill jobs channel periodically with urls to ping
	wg.Add(1) //wait for job refiller
	go scheduler.JobHandler(jobs, config.ProdConfig.URLs, config.ProdConfig.RateLimitPerSec, config.ProdConfig.GetRequestIntervalDuration(), finish, &wg)

	timeout := time.Duration(config.ProdConfig.RequestTimeOutSecs) * time.Second
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
	for i := 0; i < config.ProdConfig.WorkerCount; i++ {
		wg.Add(1) //wait for worker
		go pinger.Worker(i, jobs, results, permits, timeout, client, finish, &wg)

	}

	//read results channel and log outputs
	wg.Add(1) //wait for aggregator
	go aggregator.Aggregate(results, config.ProdConfig.OutputDir, finish, cancel, &wg)

	//initiate the event handler
	wg.Add(1) //wait for event handler
	go notification.EventHandler(config.ProdConfig.OutputDir, config.ProdConfig.NotificationServices, finish, cancel, &wg)

	<-finish.Done()
	wg.Wait()
	fmt.Println("Shutting down")
}
