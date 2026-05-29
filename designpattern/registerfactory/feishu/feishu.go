package feishu

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type feishuSender struct {
	webhook string
	secret  string
}

func (f *feishuSender) Send(message string) error {
	fmt.Printf("Using FeiShu send message: %v (webhook=%s)\n", message, f.webhook)
	return nil
}

// Close 实现 sender.Closer，释放连接资源。
func (f *feishuSender) Close() error {
	fmt.Println("FeiShu sender closed")
	return nil
}

// HealthCheck 实现 sender.HealthChecker，验证 webhook 可达。
func (f *feishuSender) HealthCheck() error {
	if f.webhook == "" {
		return fmt.Errorf("feishu: webhook not configured")
	}
	return nil
}

func init() {
	sender.Register("feishu", func(cfg sender.Config) (sender.Sender, error) {
		webhook := cfg["webhook"]
		if webhook == "" {
			webhook = "https://open.feishu.cn/open-apis/bot/v2/hook"
		}
		return &feishuSender{webhook: webhook, secret: cfg["secret"]}, nil
	})
}
