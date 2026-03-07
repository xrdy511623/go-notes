package message

import (
	"fmt"

	"go-notes/designpattern/bridge/channel"
)

// Message 是消息类型的抽象（桥接模式中的 Abstraction）。
// 每种消息类型负责格式化内容，然后委托给 channel.Sender 完成投递。
type Message interface {
	Send(to string, ch channel.Sender) error
}

// AlertMessage 表示一条告警消息，包含标题和严重等级。
type AlertMessage struct {
	Title    string
	Severity string // info / warn / critical
}

// NewAlert 创建一条指定标题和严重等级的告警消息。
func NewAlert(title, severity string) *AlertMessage {
	return &AlertMessage{Title: title, Severity: severity}
}

// Send 将告警消息格式化后，通过指定渠道发送给接收者。
// subject 格式: [ALERT][severity] title
func (a *AlertMessage) Send(to string, ch channel.Sender) error {
	subject := fmt.Sprintf("[ALERT][%s] %s", a.Severity, a.Title)
	body := fmt.Sprintf("告警等级: %s\n告警标题: %s\n请及时处理。", a.Severity, a.Title)
	return ch.Send(to, subject, body)
}
