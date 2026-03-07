package export

import (
	"bytes"
	"fmt"
)

// Record 表示一条待导出的数据记录。
type Record struct {
	Fields map[string]string
}

// Exporter 定义数据导出算法中可变的步骤。
//
// 这是 Go 式模板方法的核心：接口定义"什么会变"，
// Export() 函数定义"什么不变"（算法骨架）。
//
// 与 sort.Interface 的对照：
//
//	sort.Interface 定义 Len / Less / Swap  →  sort.Sort 调用它们
//	Exporter 定义 Header / FormatRecord / Footer  →  Export 调用它们
//
// 经典模板方法用继承 + 抽象方法，Go 用接口 + 自由函数——效果相同，耦合更低。
type Exporter interface {
	Header(columns []string) ([]byte, error)
	FormatRecord(index int, record Record, columns []string) ([]byte, error)
	Footer(totalRecords int) ([]byte, error)
}

// Export 是模板方法——它定义了固定的导出算法骨架：
//
//  1. 写入表头（Header）
//  2. 逐条格式化记录（FormatRecord）
//  3. 写入表尾（Footer）
//
// 可变步骤由 Exporter 接口提供。不同的 Exporter 实现（CSV、JSON、Markdown）
// 产生不同格式的输出，但算法流程完全相同。
func Export(e Exporter, columns []string, records []Record) ([]byte, error) {
	var buf bytes.Buffer

	header, err := e.Header(columns)
	if err != nil {
		return nil, fmt.Errorf("export: header: %w", err)
	}
	buf.Write(header)

	for i, rec := range records {
		line, err := e.FormatRecord(i, rec, columns)
		if err != nil {
			return nil, fmt.Errorf("export: record[%d]: %w", i, err)
		}
		buf.Write(line)
	}

	footer, err := e.Footer(len(records))
	if err != nil {
		return nil, fmt.Errorf("export: footer: %w", err)
	}
	buf.Write(footer)

	return buf.Bytes(), nil
}

// ExportFunc 允许用函数字段构建 Exporter，无需定义新类型。
//
// 这是"函数字段"变体的 Go 式模板方法，类似 http.HandlerFunc 把函数适配为接口。
// 适合一次性使用或测试场景；正式的格式导出应使用具名类型。
//
// nil 字段被安全处理：返回 nil, nil（空输出，无错误）。
type ExportFunc struct {
	HeaderFn func(columns []string) ([]byte, error)
	RecordFn func(index int, record Record, columns []string) ([]byte, error)
	FooterFn func(totalRecords int) ([]byte, error)
}

func (f *ExportFunc) Header(columns []string) ([]byte, error) {
	if f.HeaderFn == nil {
		return nil, nil
	}
	return f.HeaderFn(columns)
}

func (f *ExportFunc) FormatRecord(index int, record Record, columns []string) ([]byte, error) {
	if f.RecordFn == nil {
		return nil, nil
	}
	return f.RecordFn(index, record, columns)
}

func (f *ExportFunc) Footer(totalRecords int) ([]byte, error) {
	if f.FooterFn == nil {
		return nil, nil
	}
	return f.FooterFn(totalRecords)
}
