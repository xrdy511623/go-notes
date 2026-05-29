package weixin

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type weixinSender struct {
	webhook string
	key     string
}

func (w *weixinSender) Send(message string) error {
	fmt.Printf("Using WeiXin send message: %v (webhook=%s)\n", message, w.webhook)
	return nil
}

// Close 实现 sender.Closer，释放连接资源。
func (w *weixinSender) Close() error {
	fmt.Println("WeiXin sender closed")
	return nil
}

// HealthCheck 实现 sender.HealthChecker，验证 webhook 可达。
func (w *weixinSender) HealthCheck() error {
	if w.webhook == "" {
		return fmt.Errorf("weixin: webhook not configured")
	}
	return nil
}

func init() {
	sender.Register("weixin", func(cfg sender.Config) (sender.Sender, error) {
		webhook := cfg["webhook"]
		if webhook == "" {
			webhook = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"
		}
		return &weixinSender{webhook: webhook, key: cfg["key"]}, nil
	})
}
