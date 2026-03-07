package logadapter

import (
	"fmt"
	"log"
	"log/slog"
)

// Logger 是应用层的日志接口。
// 不同的日志库有不同的 API（stdlib log 用 Printf、slog 用结构化参数、
// zap 用 Sugar/Desugar），业务代码不应该绑定到任何一种具体实现。
//
// 适配器模式的作用：让多种不兼容的日志库都能适配到这个统一接口，
// 业务代码只依赖 Logger 接口，切换日志库时只需替换适配器。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// --- 适配器 1：stdlib *log.Logger → Logger ---

// StdAdapter 将标准库 *log.Logger 适配为 Logger 接口。
// *log.Logger 只提供 Printf/Println 等方法，没有 Debug/Info/Error 分级——
// 适配器在消息前添加级别标签来弥补这一差距。
type StdAdapter struct {
	l *log.Logger
}

// FromStd 创建一个将 *log.Logger 适配为 Logger 的适配器。
func FromStd(l *log.Logger) *StdAdapter {
	return &StdAdapter{l: l}
}

func (a *StdAdapter) Debug(msg string, args ...any) {
	a.l.Printf("[DEBUG] "+msg, args...)
}

func (a *StdAdapter) Info(msg string, args ...any) {
	a.l.Printf("[INFO] "+msg, args...)
}

func (a *StdAdapter) Error(msg string, args ...any) {
	a.l.Printf("[ERROR] "+msg, args...)
}

// --- 适配器 2：*slog.Logger → Logger ---

// SlogAdapter 将 Go 1.21+ 的 *slog.Logger 适配为 Logger 接口。
// slog 使用结构化日志（key-value pairs），而 Logger 接口使用 printf 风格——
// 适配器通过 fmt.Sprintf 预格式化消息，桥接两种不同的日志范式。
type SlogAdapter struct {
	l *slog.Logger
}

// FromSlog 创建一个将 *slog.Logger 适配为 Logger 的适配器。
func FromSlog(l *slog.Logger) *SlogAdapter {
	return &SlogAdapter{l: l}
}

func (a *SlogAdapter) Debug(msg string, args ...any) {
	a.l.Debug(fmt.Sprintf(msg, args...))
}

func (a *SlogAdapter) Info(msg string, args ...any) {
	a.l.Info(fmt.Sprintf(msg, args...))
}

func (a *SlogAdapter) Error(msg string, args ...any) {
	a.l.Error(fmt.Sprintf(msg, args...))
}

// --- 空实现：测试用 ---

// Nop 返回一个丢弃所有日志的 Logger，用于测试或禁用日志的场景。
func Nop() Logger { return nopLogger{} }

type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}
