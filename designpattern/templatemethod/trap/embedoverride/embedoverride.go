package embedoverride

// ❌ Java/C++ 开发者的本能：用 struct embedding 模拟继承来实现模板方法。
//
// 在 Java 中，模板方法通过继承实现：
//
//	abstract class Base {
//	    final void export(String data) { // 模板方法
//	        String formatted = format(data); // 调用虚方法
//	        System.out.println(formatted);
//	    }
//	    abstract String format(String data); // 子类覆写
//	}
//
// Go 开发者有时尝试用 embedding 复刻——但 embedding 不是继承。

// BaseExporter 提供"默认"的 Format 和 Export。
type BaseExporter struct{}

func (b *BaseExporter) Format(data string) string {
	return "base:" + data
}

// Export 是"模板方法"：固定算法骨架，调用 Format 作为可变步骤。
func (b *BaseExporter) Export(data string) string {
	formatted := b.Format(data) // ← 永远调用 BaseExporter.Format
	return "HEADER\n" + formatted + "\nFOOTER"
}

// CSVExporter 嵌入 BaseExporter，期望"覆写" Format。
type CSVExporter struct {
	BaseExporter
}

// Format 看起来像是覆写了 BaseExporter.Format，但实际上不是。
func (c *CSVExporter) Format(data string) string {
	return "csv:" + data
}

// Demo 展示 embedding 不能实现虚方法调度。
//
// 调用 csv.Export("hello") 时：
//  1. CSVExporter 没有自己的 Export 方法
//  2. Go 通过 embedding 提升（promote）了 BaseExporter.Export
//  3. BaseExporter.Export 内部调用 b.Format(data)
//  4. b 的类型是 *BaseExporter（不是 *CSVExporter）
//  5. 所以调用的是 BaseExporter.Format，不是 CSVExporter.Format
//
// 结果："HEADER\nbase:hello\nFOOTER"
// 期望："HEADER\ncsv:hello\nFOOTER"
//
// 根因：Go 没有虚方法表（vtable），方法接收者在编译期确定。
// Embedding 是字段提升，不是继承。
//
// ✅ 正确做法：用 interface + free function（本项目的 export.Export + Exporter 接口）。
func Demo() string {
	csv := &CSVExporter{}
	return csv.Export("hello")
}
