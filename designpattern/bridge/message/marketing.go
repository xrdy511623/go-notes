package message

import (
	"fmt"

	"go-notes/designpattern/bridge/channel"
)

// MarketingMessage 表示一条营销推广消息。
type MarketingMessage struct {
	Campaign string
	Content  string
}

// NewMarketing 创建一条指定活动名称和内容的营销消息。
func NewMarketing(campaign, content string) *MarketingMessage {
	return &MarketingMessage{Campaign: campaign, Content: content}
}

// Send 将营销消息格式化后，通过指定渠道发送给接收者。
// 自动追加退订页脚。
func (m *MarketingMessage) Send(to string, ch channel.Sender) error {
	subject := m.Campaign
	body := fmt.Sprintf("%s\n\n---\nTo unsubscribe, reply STOP.", m.Content)
	return ch.Send(to, subject, body)
}
