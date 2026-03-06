package logger

import (
	"fmt"

	"go-notes/designpattern/observer/eventbus"
)

// LogObserver 将接收到的事件打印到标准输出。
type LogObserver struct {
	prefix string
}

// New 创建一个带前缀的日志观察者。
func New(prefix string) *LogObserver {
	return &LogObserver{prefix: prefix}
}

func (l *LogObserver) OnEvent(event eventbus.Event) {
	fmt.Printf("[%s] topic=%s payload=%v\n", l.prefix, event.Topic, event.Payload)
}

func (l *LogObserver) Name() string {
	return "logger:" + l.prefix
}
