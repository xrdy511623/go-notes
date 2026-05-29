package dingding

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type dingSender struct {
	webhook string
	token   string
}

func (d *dingSender) Send(message string) error {
	fmt.Printf("Using DingDing send message: %v (webhook=%s)\n", message, d.webhook)
	return nil
}

// Close 实现 sender.Closer，释放连接资源。
func (d *dingSender) Close() error {
	fmt.Println("DingDing sender closed")
	return nil
}

// HealthCheck 实现 sender.HealthChecker，验证 webhook 可达。
func (d *dingSender) HealthCheck() error {
	if d.webhook == "" {
		return fmt.Errorf("dingding: webhook not configured")
	}
	return nil
}

func init() {
	sender.Register("dingding", func(cfg sender.Config) (sender.Sender, error) {
		webhook := cfg["webhook"]
		if webhook == "" {
			webhook = "https://oapi.dingtalk.com/robot/send"
		}
		return &dingSender{webhook: webhook, token: cfg["token"]}, nil
	})
}
