package channel

import (
	"fmt"
	"sync"
)

// SMSSender 通过短信网关投递消息。
type SMSSender struct {
	gateway string
	sent    []Record
	mu      sync.Mutex
}

// NewSMS 创建一个指定网关的短信渠道。
func NewSMS(gateway string) *SMSSender {
	return &SMSSender{gateway: gateway}
}

// Send 向指定号码发送短信。to 不能为空。
func (s *SMSSender) Send(to, subject, body string) error {
	if to == "" {
		return fmt.Errorf("channel/sms: recipient must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, Record{To: to, Subject: subject, Body: body})
	return nil
}

// Sent 返回已发送记录的副本（并发安全）。
func (s *SMSSender) Sent() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]Record, len(s.sent))
	copy(cp, s.sent)
	return cp
}
