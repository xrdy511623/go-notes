package notifier

import (
	"fmt"

	"go-notes/designpattern/observer/eventbus"
)

// NotifyObserver 模拟通过指定渠道发送通知。
type NotifyObserver struct {
	channel string
}

// New 创建一个指定渠道的通知观察者。
func New(channel string) *NotifyObserver {
	return &NotifyObserver{channel: channel}
}

func (n *NotifyObserver) OnEvent(event eventbus.Event) {
	fmt.Printf("Notify [%s]: topic=%s payload=%v\n", n.channel, event.Topic, event.Payload)
}

func (n *NotifyObserver) Name() string {
	return "notifier:" + n.channel
}
