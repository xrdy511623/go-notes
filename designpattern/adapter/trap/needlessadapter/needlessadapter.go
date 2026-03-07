package needlessadapter

import "io"

// ❌ 错误示例：为已经满足接口的类型创建多余的适配器
//
// Go 的隐式接口（structural typing）意味着：只要类型有正确的方法签名，
// 它就自动满足接口——不需要 implements 声明，更不需要手写适配器包装。
// 这是 Go 与 Java/C# 最大的区别之一。
//
// 从 Java 转来的开发者最常犯的错误：看到"类型 A 要满足接口 B"，
// 本能地写一个 Adapter struct 做转发。在 Go 中，如果 A 已经有了 B 要求
// 的方法，直接用就行——编译器会自动匹配。

// FileLogger 是一个自定义的日志写入器。
type FileLogger struct {
	Path string
}

// Write 实现了 io.Writer 接口——Go 编译器自动识别。
func (f *FileLogger) Write(p []byte) (int, error) {
	// 实际写入逻辑省略
	return len(p), nil
}

// ❌ FileLoggerWriterAdapter 是完全多余的。
// FileLogger 已经有了 Write([]byte) (int, error) 方法，
// 它天然就是 io.Writer——不需要任何适配器。
//
// 这种"为了适配而适配"的代码在 Java 中是必须的（因为 Java 需要 implements），
// 但在 Go 中只是增加了一层无意义的间接调用。
type FileLoggerWriterAdapter struct {
	logger *FileLogger
}

func NewFileLoggerWriterAdapter(l *FileLogger) *FileLoggerWriterAdapter {
	return &FileLoggerWriterAdapter{logger: l}
}

func (a *FileLoggerWriterAdapter) Write(p []byte) (int, error) {
	return a.logger.Write(p) // ❌ 纯粹的转发，零附加值
}

// ✅ 正确做法：直接使用 FileLogger，Go 的隐式接口处理一切。
func CorrectUsage() io.Writer {
	return &FileLogger{Path: "/var/log/app.log"}
}

// ❌ WrongUsage 多了一层毫无意义的包装。
func WrongUsage() io.Writer {
	return NewFileLoggerWriterAdapter(&FileLogger{Path: "/var/log/app.log"})
}
