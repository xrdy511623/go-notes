package export

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

// stubExporter produces predictable output for testing the template algorithm.
type stubExporter struct {
	headerErr error
	recordErr error
	footerErr error
}

func (s *stubExporter) Header(columns []string) ([]byte, error) {
	if s.headerErr != nil {
		return nil, s.headerErr
	}
	return []byte("H:" + strings.Join(columns, ",") + "\n"), nil
}

func (s *stubExporter) FormatRecord(index int, record Record, columns []string) ([]byte, error) {
	if s.recordErr != nil {
		return nil, s.recordErr
	}
	vals := make([]string, len(columns))
	for i, c := range columns {
		vals[i] = record.Fields[c]
	}
	return []byte("R:" + strings.Join(vals, ",") + "\n"), nil
}

func (s *stubExporter) Footer(total int) ([]byte, error) {
	if s.footerErr != nil {
		return nil, s.footerErr
	}
	return []byte("F:" + strconv.Itoa(total) + "\n"), nil
}

func records(rows ...map[string]string) []Record {
	recs := make([]Record, len(rows))
	for i, r := range rows {
		recs[i] = Record{Fields: r}
	}
	return recs
}

func TestExport_BasicOutput(t *testing.T) {
	data := records(
		map[string]string{"name": "Alice", "age": "30"},
		map[string]string{"name": "Bob", "age": "25"},
	)
	out, err := Export(&stubExporter{}, []string{"name", "age"}, data)
	if err != nil {
		t.Fatal(err)
	}
	expected := "H:name,age\nR:Alice,30\nR:Bob,25\nF:2\n"
	if string(out) != expected {
		t.Fatalf("got:\n%s\nwant:\n%s", out, expected)
	}
}

func TestExport_AlgorithmOrder(t *testing.T) {
	var order []string
	e := &ExportFunc{
		HeaderFn: func([]string) ([]byte, error) {
			order = append(order, "header")
			return nil, nil
		},
		RecordFn: func(int, Record, []string) ([]byte, error) {
			order = append(order, "record")
			return nil, nil
		},
		FooterFn: func(int) ([]byte, error) {
			order = append(order, "footer")
			return nil, nil
		},
	}
	Export(e, []string{"a"}, records(map[string]string{"a": "1"}, map[string]string{"a": "2"}))

	expected := []string{"header", "record", "record", "footer"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range order {
		if v != expected[i] {
			t.Errorf("step[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestExport_EmptyRecords(t *testing.T) {
	out, err := Export(&stubExporter{}, []string{"x"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := "H:x\nF:0\n"
	if string(out) != expected {
		t.Fatalf("got %q, want %q", out, expected)
	}
}

func TestExport_ColumnOrder(t *testing.T) {
	data := records(map[string]string{"b": "2", "a": "1", "c": "3"})
	out, _ := Export(&stubExporter{}, []string{"c", "a", "b"}, data)

	if !strings.Contains(string(out), "R:3,1,2") {
		t.Fatalf("expected columns in c,a,b order, got:\n%s", out)
	}
}

func TestExport_MissingField(t *testing.T) {
	data := records(map[string]string{"name": "Alice"})
	out, _ := Export(&stubExporter{}, []string{"name", "missing"}, data)

	if !strings.Contains(string(out), "R:Alice,") {
		t.Fatalf("missing field should produce empty string, got:\n%s", out)
	}
}

func TestExport_HeaderError(t *testing.T) {
	e := &stubExporter{headerErr: errors.New("bad header")}
	_, err := Export(e, []string{"a"}, nil)
	if err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("expected header error, got %v", err)
	}
}

func TestExport_RecordError(t *testing.T) {
	e := &stubExporter{recordErr: errors.New("bad record")}
	_, err := Export(e, []string{"a"}, records(map[string]string{"a": "1"}))
	if err == nil || !strings.Contains(err.Error(), "record[0]") {
		t.Fatalf("expected record error with index, got %v", err)
	}
}

func TestExport_FooterError(t *testing.T) {
	e := &stubExporter{footerErr: errors.New("bad footer")}
	_, err := Export(e, []string{"a"}, nil)
	if err == nil || !strings.Contains(err.Error(), "footer") {
		t.Fatalf("expected footer error, got %v", err)
	}
}

func TestExportFunc_Basic(t *testing.T) {
	e := &ExportFunc{
		HeaderFn: func(cols []string) ([]byte, error) {
			return []byte("[" + strings.Join(cols, "|") + "]"), nil
		},
		RecordFn: func(_ int, r Record, cols []string) ([]byte, error) {
			return []byte("{" + r.Fields[cols[0]] + "}"), nil
		},
		FooterFn: func(n int) ([]byte, error) {
			return []byte("=" + strconv.Itoa(n)), nil
		},
	}
	out, err := Export(e, []string{"x"}, records(map[string]string{"x": "hello"}))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "[x]{hello}=1" {
		t.Fatalf("got %q", out)
	}
}

func TestExportFunc_NilFunctions(t *testing.T) {
	e := &ExportFunc{}
	out, err := Export(e, []string{"a"}, records(map[string]string{"a": "1"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty output for nil functions, got %q", out)
	}
}

func TestExportFunc_PartialNil(t *testing.T) {
	e := &ExportFunc{
		HeaderFn: func([]string) ([]byte, error) { return []byte("HEAD"), nil },
	}
	out, err := Export(e, []string{"a"}, records(map[string]string{"a": "1"}))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "HEAD" {
		t.Fatalf("expected only header, got %q", out)
	}
}

func TestExport_RecordIndex(t *testing.T) {
	var indices []int
	e := &ExportFunc{
		RecordFn: func(i int, _ Record, _ []string) ([]byte, error) {
			indices = append(indices, i)
			return nil, nil
		},
	}
	Export(e, []string{"a"}, records(
		map[string]string{"a": "1"},
		map[string]string{"a": "2"},
		map[string]string{"a": "3"},
	))

	expected := []int{0, 1, 2}
	if len(indices) != len(expected) {
		t.Fatalf("expected %d indices, got %d", len(expected), len(indices))
	}
	for i, idx := range indices {
		if idx != expected[i] {
			t.Errorf("index[%d] = %d, want %d", i, idx, expected[i])
		}
	}
}

func TestExport_LargeDataset(t *testing.T) {
	data := make([]Record, 1000)
	for i := range data {
		data[i] = Record{Fields: map[string]string{"id": strconv.Itoa(i)}}
	}
	out, err := Export(&stubExporter{}, []string{"id"}, data)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// 1 header + 1000 records + 1 footer = 1002
	if len(lines) != 1002 {
		t.Fatalf("expected 1002 lines, got %d", len(lines))
	}
}

func TestExport_SatisfiesInterface(t *testing.T) {
	var _ Exporter = &stubExporter{}
	var _ Exporter = &ExportFunc{}
}
