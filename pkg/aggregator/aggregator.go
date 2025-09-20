package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/scrapper"
	"go.uber.org/zap"
)

func Aggregate(results chan scrapper.ScrapeResult, outputDir string, log *zap.Logger, finish context.Context, cancel context.CancelFunc) {
	filename := fmt.Sprintf("gsm-%s.json", time.Now().Format("20060102_150405"))
	file, err := os.OpenFile(path.Join(outputDir,filename), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Error("Result file Error")
		cancel()
	}
	defer file.Close()
	for {
		select {
		case <-finish.Done():
			return
		case res := <-results:
			switch {
			case res.Status == -1:
				log.Error("Ping Failed",
					zap.String("URL", res.URL),
					zap.String("TimeStampUTC", res.TimestampUTC),
					zap.String("Error", res.Error),
					zap.Int("WorkerID", res.WorkerID))
			case res.Status >= 400:
				log.Warn("Non-2XX Status",
					zap.String("URL", res.URL),
					zap.String("TimeStampUTC", res.TimestampUTC),
					zap.Int("Status", res.Status),
					zap.Int("Latency", int(res.ResponseMS)),
					zap.Int("WorkerID", res.WorkerID))
			default:
				log.Info("Ping Success",
					zap.String("URL", res.URL),
					zap.String("TimeStampUTC", res.TimestampUTC),
					zap.Int("Status", res.Status),
					zap.Int("Latency", int(res.ResponseMS)),
					zap.Int("WorkerID", res.WorkerID))
			}
			line, err := json.Marshal(res)
			if err != nil {
				log.Error("Result unparsable")
				cancel()
			}
			_,err = file.Write(append(line,'\n'))
			if err != nil {
				log.Error("Result file write error")
				cancel()
			}
		}
	}
}
