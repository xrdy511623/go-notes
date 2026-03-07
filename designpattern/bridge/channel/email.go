package channel

import (
	"fmt"
	"strings"
	"sync"
)

// Record 表示一条已发送的消息记录，用于测试验证。
type Record struct {
	To      string
	Subject string
	Body    string
}

// Sender 是消息投递渠道的抽象（桥接模式中的 Implementor）。
// 所有具体渠道（邮件、短信、Webhook）都必须实现此接口。
type Sender interface {
	Send(to, subject, body string) error
}

// EmailSender 通过电子邮件投递消息。
type EmailSender struct {
	from string
	sent []Record
	mu   sync.Mutex
}

// NewEmail 创建一个以 from 为发件人的邮件渠道。
func NewEmail(from string) *EmailSender {
	return &EmailSender{from: from}
}

// Send 向指定收件人发送邮件。to 必须包含 "@" 符号。
func (e *EmailSender) Send(to, subject, body string) error {
	if !strings.Contains(to, "@") {
		return fmt.Errorf("channel/email: invalid email address %q", to)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.sent = append(e.sent, Record{To: to, Subject: subject, Body: body})
	return nil
}

// Sent 返回已发送记录的副本（并发安全）。
func (e *EmailSender) Sent() []Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]Record, len(e.sent))
	copy(cp, e.sent)
	return cp
}
