package notifier

import (
	"fmt"
	"sync"
)

// Message 表示一条待发送的通知消息。
type Message struct {
	To      string
	Subject string
	Body    string
}

// Service 负责发送通知并记录已发送的消息，便于测试验证。
type Service struct {
	mu   sync.Mutex
	sent []Message
}

// New 创建一个新的通知服务实例。
func New() *Service {
	return &Service{}
}

// Send 发送一条通知消息。To、Subject、Body 均不能为空。
func (s *Service) Send(msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("notifier: To must not be empty")
	}
	if msg.Subject == "" {
		return fmt.Errorf("notifier: Subject must not be empty")
	}
	if msg.Body == "" {
		return fmt.Errorf("notifier: Body must not be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, msg)
	return nil
}

// Sent 返回已发送消息的副本，用于测试断言。
func (s *Service) Sent() []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]Message, len(s.sent))
	copy(cp, s.sent)
	return cp
}
