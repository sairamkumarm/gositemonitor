package scheduler

import (
	"context"
	"time"
)

func PermitHandler(permits chan struct{}, rateLimitPerSec int, finish context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(rateLimitPerSec))
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
}

func JobHandler(jobs chan string, urls []string, rateLimitPerSec int, requestIntervalDuration time.Duration, finish context.Context){
	ticker := time.NewTicker(requestIntervalDuration)
		defer ticker.Stop()
	mainloop:
		for {
			for _, url := range urls {
				for range rateLimitPerSec {
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
}