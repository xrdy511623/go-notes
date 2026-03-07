package channel

import (
	"fmt"
	"sync"
)

// WebhookSender 通过 HTTP Webhook 投递消息。
type WebhookSender struct {
	url  string
	sent []Record
	mu   sync.Mutex
}

// NewWebhook 创建一个指定 URL 的 Webhook 渠道。
func NewWebhook(url string) *WebhookSender {
	return &WebhookSender{url: url}
}

// Send 向 Webhook 端点投递消息。url 不能为空。
func (w *WebhookSender) Send(to, subject, body string) error {
	if w.url == "" {
		return fmt.Errorf("channel/webhook: url must not be empty")
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = append(w.sent, Record{To: to, Subject: subject, Body: body})
	return nil
}

// Sent 返回已发送记录的副本（并发安全）。
func (w *WebhookSender) Sent() []Record {
	w.mu.Lock()
	defer w.mu.Unlock()
	cp := make([]Record, len(w.sent))
	copy(cp, w.sent)
	return cp
}
