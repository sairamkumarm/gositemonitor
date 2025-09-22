package notification

import "github.com/sairamkumarm/gositemonitor/pkg/logger"

func sendNotifications(event Event, senders []string) {
	for _, name := range senders {
		sender, ok := NotificationSenders[name]
		if !ok {
			logger.Log.Warn("Unknowen Service")
		} else {
			err := sender.Send(event)
			if err != nil {
				logger.Log.Error("Error Sending to "+name)
			}
		}
	}
}
