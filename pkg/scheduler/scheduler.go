package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func PermitHandler(permits chan struct{}, rateLimitPerSec int, finish context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(time.Second / time.Duration(rateLimitPerSec))
	defer ticker.Stop()
mainloop:
	for {
		<-ticker.C
		select {
		case <-finish.Done():
			fmt.Println("Deactivating Permit Handler")
			break mainloop
		case permits <- struct{}{}:
		}
	}
}

func JobHandler(jobs chan string, urls []string, rateLimitPerSec int, requestIntervalDuration time.Duration, finish context.Context, wg *sync.WaitGroup) {
	defer func() {
		fmt.Println("Deactivating Job Refiller")
		wg.Done()
	}()
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
		select {
		case <-finish.Done():
			break mainloop
		case <-ticker.C:
		}

	}
}
