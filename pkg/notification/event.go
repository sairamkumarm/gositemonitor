package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"go.uber.org/zap"
)

type Notifiable any

type Event struct {
	Message string
	Data Notifiable
	TimestampUTC time.Time
}

var EventChannel = make(chan Event, 100)

func EventHandler(outputDir string,notificationServices []string, finish context.Context,  cancel context.CancelFunc, wg *sync.WaitGroup){
	defer wg.Done()
	filename := fmt.Sprintf("gsm-%s-events.json", time.Now().Format("20060102_150405"))
	file, err := os.OpenFile(path.Join(outputDir,filename), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logger.Log.Error("Result file Error", zap.Error(err))
		cancel()
	}
	defer file.Close()
	for{
		select{
		case <-finish.Done():
			fmt.Println("Deactivating Notification Handler")
			return;
		case notif:= <- EventChannel:
			//do something with the notification
			sendNotifications(notif,notificationServices)
			bytes, err := json.Marshal(notif)
			if err != nil {
				logger.Log.Error("notification unparsable", zap.Error(err))
				cancel()
			}
			_,err = file.Write(append(bytes, '\n'))
			if err != nil {
				logger.Log.Error("event file write error", zap.Error(err))
			}
		}
	}
}