package jsonexport

import (
	"encoding/json"

	"go-notes/designpattern/templatemethod/export"
)

// Exporter 将数据导出为 JSON 数组格式。
// 利用 index 参数处理元素间的逗号——这是模板方法传递上下文信息的典型用法。
type Exporter struct{}

// New 创建一个 JSON 导出器。
func New() *Exporter {
	return &Exporter{}
}

func (e *Exporter) Header(_ []string) ([]byte, error) {
	return []byte("[\n"), nil
}

func (e *Exporter) FormatRecord(index int, record export.Record, columns []string) ([]byte, error) {
	obj := make(map[string]string, len(columns))
	for _, col := range columns {
		obj[col] = record.Fields[col]
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if index > 0 {
		return append([]byte(",\n  "), data...), nil
	}
	return append([]byte("  "), data...), nil
}

func (e *Exporter) Footer(_ int) ([]byte, error) {
	return []byte("\n]\n"), nil
}
