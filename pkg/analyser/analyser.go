package analyser

import (
	"context"
	"time"

	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"github.com/sairamkumarm/gositemonitor/pkg/notification"
	"github.com/sairamkumarm/gositemonitor/pkg/scrapper"
	"go.uber.org/zap"
)

type Stat struct {
	Url              string
	OutageStart      time.Time
	OutageLatest     time.Time
	ConsecutiveFails int
	TotalFails       int
	MaxLatency       int64
}

var Stats = make(map[string]*Stat)

func FillInitialUrls(urls []string) {
	for _, url := range urls {
		Stats[url] = &Stat{Url: url}
	}
}

func AnalyseResult(res scrapper.ScrapeResult, finish context.Context) {
	_, ok := Stats[res.URL]
	if ok {
		stat := Stats[res.URL]
		if res.Status == -1 || res.Status >= 400 {
			if stat.OutageStart.IsZero() { //first error, possible start of outage
				stat.OutageStart= res.TimestampUTC
			}
			stat.OutageLatest= res.TimestampUTC //latest time of outage
			stat.ConsecutiveFails++
			stat.TotalFails++
			if stat.ConsecutiveFails == 3 {
				logger.Log.Error("Possible outage in progress", zap.Any("outage", stat))
				//log and write to notification channel about ongoing outage
				statcopy := *stat
				notif := notification.Event{Message: "Possible outage in progress", Data: statcopy, TimestampUTC: time.Now()}
				select{
				case <-finish.Done():
					return
				case notification.EventChannel <- notif:
					//safe enqueue
				}
			}
		} else {
			//non error, two possibilies, outage recovered, normal success
			if !stat.OutageStart.IsZero() { //this is the conclusion of a previous outage
				stat.OutageLatest= res.TimestampUTC
				//log and write to a notification channel about outage
				logger.Log.Warn("Outage report", zap.Any("outage", stat))
				statcopy := *stat
				notif:= notification.Event{Message: "Outage Report",Data: statcopy, TimestampUTC: time.Now()}
				select{
				case <-finish.Done():
					return
				case notification.EventChannel <- notif:
					//safe enqueue
				}
				//reseting values except total
				stat.ConsecutiveFails = 0
				stat.OutageLatest = time.Time{} //sets time.Time to zero value
				stat.OutageStart = time.Time{}
			}
			stat.MaxLatency = max(stat.MaxLatency, res.ResponseMS)
		}
	}
}
