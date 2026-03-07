package notifier

import (
	"go-notes/designpattern/bridge/channel"
	"go-notes/designpattern/bridge/message"
)

// Notifier 使用桥接模式组合消息类型与投递渠道。
// 它是一个便利层，持有默认渠道并提供广播能力。
type Notifier struct {
	defaultChannel channel.Sender
}

// New 创建一个使用默认渠道的 Notifier。
func New(ch channel.Sender) *Notifier {
	return &Notifier{defaultChannel: ch}
}

// Notify 通过默认渠道发送消息。
func (n *Notifier) Notify(to string, msg message.Message) error {
	return msg.Send(to, n.defaultChannel)
}

// NotifyVia 通过指定渠道发送消息，覆盖默认渠道。
func (n *Notifier) NotifyVia(to string, msg message.Message, ch channel.Sender) error {
	return msg.Send(to, ch)
}

// Broadcast 向多个接收者通过默认渠道发送消息。
// 返回值仅包含发送失败的接收者及其错误；全部成功时返回 nil。
func (n *Notifier) Broadcast(recipients []string, msg message.Message) map[string]error {
	var errs map[string]error
	for _, to := range recipients {
		if err := msg.Send(to, n.defaultChannel); err != nil {
			if errs == nil {
				errs = make(map[string]error)
			}
			errs[to] = err
		}
	}
	return errs
}
