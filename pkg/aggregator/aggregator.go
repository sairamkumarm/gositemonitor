package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/analyser"
	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"github.com/sairamkumarm/gositemonitor/pkg/scrapper"
	"go.uber.org/zap"
)

func Aggregate(results chan scrapper.ScrapeResult, outputDir string, finish context.Context, cancel context.CancelFunc, wg *sync.WaitGroup) {
	defer wg.Done()
	filename := fmt.Sprintf("gsm-%s.json", time.Now().Format("20060102_150405"))
	file, err := os.OpenFile(path.Join(outputDir,filename), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Log.Error("Result file Error", zap.Error(err))
		cancel()
	}
	defer file.Close()
	for {
		select {
		case <-finish.Done():
			fmt.Println("Deactivating Aggregator")
			return
		case res := <-results:
			//write logs of result
			logger.ResultLogger(res)

			//send result for analysis and notification down the line
			go analyser.AnalyseResult(res, finish)


			line, err := json.Marshal(res)
			if err != nil {
				logger.Log.Error("Result unparsable", zap.Error(err))
				cancel()
			}
			_,err = file.Write(append(line,'\n'))
			if err != nil {
				logger.Log.Error("Result file write error", zap.Error(err))
				cancel()
			}
		}
	}
}
