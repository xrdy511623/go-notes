package mdexport

import (
	"fmt"
	"strings"

	"go-notes/designpattern/templatemethod/export"
)

// Exporter 将数据导出为 Markdown 表格格式。
type Exporter struct{}

// New 创建一个 Markdown 表格导出器。
func New() *Exporter {
	return &Exporter{}
}

func (e *Exporter) Header(columns []string) ([]byte, error) {
	header := "| " + strings.Join(columns, " | ") + " |\n"
	sep := "|"
	for range columns {
		sep += " --- |"
	}
	return []byte(header + sep + "\n"), nil
}

func (e *Exporter) FormatRecord(_ int, record export.Record, columns []string) ([]byte, error) {
	values := make([]string, len(columns))
	for i, col := range columns {
		values[i] = record.Fields[col]
	}
	return []byte("| " + strings.Join(values, " | ") + " |\n"), nil
}

func (e *Exporter) Footer(total int) ([]byte, error) {
	return []byte(fmt.Sprintf("\n*%d records*\n", total)), nil
}
