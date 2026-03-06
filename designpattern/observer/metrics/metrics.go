package metrics

import (
	"sync/atomic"

	"go-notes/designpattern/observer/eventbus"
)

// MetricsObserver 统计接收到的事件数量。
type MetricsObserver struct {
	count int64
	label string
}

// New 创建一个带标签的指标观察者。
func New(label string) *MetricsObserver {
	return &MetricsObserver{label: label}
}

func (m *MetricsObserver) OnEvent(_ eventbus.Event) {
	atomic.AddInt64(&m.count, 1)
}

func (m *MetricsObserver) Name() string {
	return "metrics:" + m.label
}

// Count 返回已接收的事件总数。
func (m *MetricsObserver) Count() int64 {
	return atomic.LoadInt64(&m.count)
}
