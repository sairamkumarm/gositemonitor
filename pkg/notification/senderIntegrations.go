package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/mailersend/mailersend-go"
	"github.com/sairamkumarm/gositemonitor/pkg/config"
	"github.com/sairamkumarm/gositemonitor/pkg/logger"
	"go.uber.org/zap"
)

type NotificationSender interface {
	Name() string
	Send(event Event) error
}

var NotificationSenders = map[string]NotificationSender{
	"discord": &DiscordNotificationSender{name: "discord"},
	"email": &EmailNotificationSender{name: "email",
		fromName: "GoSiteMonitor",
		toName:   "GoSiteMonitorUser",
	}}

var timeout context.Context
var reqcancel context.CancelFunc

var messageClient *http.Client = &http.Client{
	Transport: &http.Transport{
		IdleConnTimeout:       1000 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second}}

type DiscordNotificationSender struct {
	name string
	webhook string
}

func (d *DiscordNotificationSender) Name() string {
	return d.name
}

func (d *DiscordNotificationSender) Send(event Event) error {
	// logger.Log.Info("Discord Alert", zap.Any("notification", event))
	d.webhook = config.ProdConfig.DiscordWebhookAddress
	timeout, reqcancel = context.WithTimeout(context.Background(), time.Second*10)
	defer reqcancel()
	data,err:=json.MarshalIndent(event.Data,""," ")
	if err != nil {
		return fmt.Errorf("json format error: %w",err)
	}
	payload:=map[string]any{"content":fmt.Sprintf("**%s**\n```json\n%s\n```",event.Message,string(data))}
	body,err:=json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json format error: %w",err)
	}
	req, err:=http.NewRequestWithContext(timeout,"POST",d.webhook,bytes.NewReader(body))
	if err!=nil {
		return fmt.Errorf("request creation error: %w",err)
	}
	req.Header.Set("Content-Type","application/json")
	resp, err := messageClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord request error: %w",err)
	}
	defer resp.Body.Close()
	b,_:=io.ReadAll(resp.Body)
	logger.Log.Info("discord resp",zap.ByteString("response",b),  zap.String("",resp.Status))
	return nil
}

type EmailNotificationSender struct {
	name        string
	apiToken    string
	fromAddress string
	fromName    string
	toAddress   string
	toName      string
}

func (e *EmailNotificationSender) Name() string {
	return e.name
}
func (e *EmailNotificationSender) Send(event Event) error {
	e.apiToken = config.ProdConfig.MailerSendAPIToken
	e.fromAddress = config.ProdConfig.MailerSendEmailId
	e.toAddress = config.ProdConfig.NotificationMailId

	timeout, reqcancel = context.WithTimeout(context.Background(), time.Second*10)
	defer reqcancel()

	ms:=mailersend.NewMailersend(e.apiToken)
	subject := event.Message
	text:="Find the details below"
	data,err:=json.MarshalIndent(event.Data,""," ")
	if err != nil {
		return fmt.Errorf("json data error: %w",err)
	}
	html:=fmt.Sprintf("<pre>%s</pre>",html.EscapeString(string(data)))
	from:=mailersend.From{
		Name: e.fromName,
		Email: e.fromAddress}
	to:=[]mailersend.Recipient{{Name:e.toName,Email:e.toAddress}}
	tags:=[]string{"GoSiteMonitor","Alert"}
	message:=ms.Email.NewMessage()
	message.SetFrom(from)
	message.SetRecipients(to)
	message.SetSubject(subject)
	message.SetText(text)
	message.SetHTML(html)
	message.SetTags(tags)
	res, err := ms.Email.Send(timeout,message)
	if err != nil {
		return fmt.Errorf("mail API error: %w",err)
	}
	defer res.Body.Close()
	if res.Status == "202 Accepted" {
		logger.Log.Info("Mail delivered")
	} else {
		logger.Log.Error("Mail delivery failed")
	}
	return nil
}
