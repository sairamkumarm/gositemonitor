package scrapper

import (
	"context"
	"net/http"
	"time"
	// "github.com/sairamkumarm/gositemonitor/pkg/logger"
)

type ScrapeResult struct {
	URL          string `json:"url"`
	Status       int    `json:"status"`
	ResponseMS   int64  `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
	TimestampUTC string `json:"timestamp_utc"`
	WorkerID     int    `json:"worker_id"`
}

func timedGet(url string, timeout time.Duration, client *http.Client) ScrapeResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ScrapeResult{
			URL:          url,
			Status:       -1,
			Error:        err.Error(),
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
		}
	}
	start := time.Now()
	res := ScrapeResult{URL: url, TimestampUTC: start.UTC().Format(time.RFC3339)}
	resp, err := client.Do(req)
	res.ResponseMS = time.Since(start).Milliseconds()
	if err != nil {
		// fmt.Println("Request Failed, ", err)
		res.Error = err.Error()
		res.Status = -1
		return res
	}
	defer resp.Body.Close()
	res.Status = resp.StatusCode
	return res
}

func Worker(id int, jobs chan string, results chan ScrapeResult, permits chan struct{}, timeoutsecs time.Duration, client *http.Client, finish context.Context) {
	for {
		select {
		case <-finish.Done():
			return
		case url, ok := <-jobs:
			if !ok {
				break
			}
			select {
			case <-finish.Done():
				return
			case _, ok = <-permits:
				if !ok {
					break
				}
			}
			res := timedGet(url, timeoutsecs, client)
			res.WorkerID = id
			select{
			case <-finish.Done():
				return
			case results <-  res:
				//nothing just enqueue
			}
		}
	}
}
