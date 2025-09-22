# GoSiteMonitor

## Overview

**GoSiteMonitor** is a concurrent site monitoring tool written in Go. It leverages a worker pool, rate limiting, and periodic job scheduling to ping URLs efficiently while respecting throughput constraints. The system is designed for observability, structured logging, and easy configuration, forming the foundation for a scalable monitoring system.

---

## Features

* **Concurrent pinging**: Configurable worker pool to process multiple URLs in parallel.
* **Rate limiting**: Global token-based throttle to control requests per second.
* **Periodic job scheduling**: Refill job queue at configurable intervals for repeated monitoring.
* **Structured logging**: JSON logs for easy aggregation and analysis.
* **Config-driven**: JSON configuration file to define URLs, worker count, rate limits, log level, request timeout, and output file.
* **Extensible results aggregation**: Centralized fan-in results channel, ready for future async logging or message broker integration.
* **Outage detection and latecy logs**: Detects patterns in ping results and logs them seperately.
* **Multi-Channel Notifications**: Sends outage alerts and reports via email and discord.
---

## Getting Started

### Prerequisites

* Go 1.20+
* Git

### Installation

```bash
git clone https://github.com/yourusername/gositemonitor.git
cd gositemonitor
go mod tidy
```

### Configuration

Create a `config.json` file in the root directory:

```json
{
  "urls": ["https://golang.org", "https://example.com"],
  "worker_count": 5,
  "rate_limit_per_sec": 2,
  "request_timeout_secs": 3,
  "request_interval":5,
  "log_level": "info",
  "output_dir": "results",
  "notification_services":["discord","email"],
  "discord_webhook_address":"https://discord.com/api/webhooks/xxxxxxxxxx/yyyyyyyyyyyyyyyy",
  "mailersend_api_token":"mlsn.xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "mailersend_email_id":"alert@xxxxxxxx.mlsender.net",
  "mail_id":"user@gmail.com"
}
```

**Parameters:**

* `urls`: List of URLs to ping.
* `worker_count`: Number of concurrent workers (minimum 5).
* `rate_limit_per_sec`: Maximum number of requests per second across all workers.
* `request_timeout_secs`: Timeout for each HTTP request.
* `request_interval` : Interval between ping job refills.
* `log_level`: Logging verbosity (`debug`, `info`, `warn`, `error`).
* `output_dir`: Directory where session based logs are stored.
* `notification_services`: Pick between discord, email or both.
* `api-tokens and keys`: Necessary to use the notification service.

---

### Running the Monitor

```bash
go run ./cmd/gositemonitor -config config.json -runtime 30
```

The monitor will:

1. Load the configuration.
2. Spawn the worker pool.
3. Refill jobs periodically according to the configured interval.
4. Log each ping result in structured JSON format.
5. Run continuously for 30 seconds or until interrupted.

---

## Architecture
```
┌────────────┐    ┌────────────────┐
│   Config   ├───▶│ Permit handler ├──▶ permit channel 
└──────┬─────┘    └────────────────┘          │           
       │                                      │
       ▼                                      │
┌──────────────┐                              │
│ Job Refiller ├───▶ jobs channel ───┐        │
└──────────────┘                     │        │
                                     ▼        ▼
                                ┌─────────────────┐
                                │    WorkerPool   │
                                │   (concurrent)  │
                                └────────┬────────┘
                                         │
                                         ▼
                                  results channel
                                         │
                                         ▼
                                ┌─────────────────┐            
                                │    Aggregator   │       ┌─────────────────────┐    
                                │   logs, stats,  │──────▶│  Write to log file  │
                                │   and patterns  │       └─────────────────────┘
                                └────────┬────────┘
                                         │
                                         ▼
                                ┌────────────────────┐
                                │      Analyser      │       
                                │  (outages reports  │
                                │and latency metrics)│       
                                └─────────┬──────────┘
                                          │
                                          ▼
                                 notification channel
                                          │
                                          ▼
                                 ┌─────────────────┐
                                 │   Notification  │
                                 │      handler    │
                                 └────────┬────────┘        ┌─────────────────────┐
                                          ├────────────────▶│ Write to event file │
                                          │                 └─────────────────────┘
                            ┌─────────────┴─────────────┐
                            │                           │
                            ▼                           ▼
                      ┌────────────┐              ┌────────────┐
                      │  Discord   │              │    Mail    │
                      └────────────┘              └────────────┘           
```
* **Job refiller**: periodically pushes jobs into `jobs` channel.
* **Worker pool**: N workers consume jobs, acquire permits, and process requests.
* **`Permits` channel**: global rate limiter controlling request throughput.
* **`Results` channel**: fan-in of ping results, consumed by aggregator for logging and future persistence.
* **Aggregator**: Listens to `results` channel, pulls results from N workers into one lane.
* **Analyser**: Finds patterns in results channel and reports to the `notification` channel.
* **Notification Handler**: Sends enriched notifications through `mail` and/or `discord` as configured.

---

## Code Structure

```
gositemonitor/
│── cmd/
│   └── gositemonitor/
│       └── main.go       # Entry point
│
├── pkg/
│   ├── config/           # JSON config loader and validation
│   ├── scheduler/        # Logic dump of routines from main.go
│   ├── pinger/           # Worker pool, ping logic
│   ├── aggregator/       # Aggregation logic
│   ├── analyser/         # finds patterns
│   ├── notification/     # sends notifications
│   └── logger/           # Zap logging setup
│
└── config.json           # Example configuration
```

---
