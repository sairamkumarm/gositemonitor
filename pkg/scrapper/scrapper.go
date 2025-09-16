package scrapper

import (
	"time"

	// "github.com/sairamkumarm/gositemonitor/pkg/logger"
)

type ScrapeResult struct {
	URL          string `json:"url"`
	Status       int    `json:"status"`
	Title        string `json:"title,omitempty"`
	ResponseMS   int64  `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
	TimestampUTC string `json:"timestamp_utc"`
	WorkerID int `json:"worker_id"`
}

func Worker(id int, jobs chan string, results chan ScrapeResult, permits chan struct{}) {
	for {
		url,ok:=<-jobs
		if !ok {break}
		<-permits
		res := ScrapeResult{url, 200, "Dummy", 345, "", time.Now().String(),id}
		results <- res
	}
}