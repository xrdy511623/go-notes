package csvexport

import (
	"strings"

	"go-notes/designpattern/templatemethod/export"
)

// Exporter 将数据导出为 CSV 格式。
// 实现 export.Exporter 接口的三个方法，由 export.Export 模板调用。
type Exporter struct {
	Separator string
}

// New 创建一个 CSV 导出器，默认使用逗号分隔。
func New() *Exporter {
	return &Exporter{Separator: ","}
}

func (e *Exporter) Header(columns []string) ([]byte, error) {
	return []byte(strings.Join(columns, e.Separator) + "\n"), nil
}

func (e *Exporter) FormatRecord(_ int, record export.Record, columns []string) ([]byte, error) {
	values := make([]string, len(columns))
	for i, col := range columns {
		values[i] = record.Fields[col]
	}
	return []byte(strings.Join(values, e.Separator) + "\n"), nil
}

func (e *Exporter) Footer(_ int) ([]byte, error) {
	return nil, nil
}
