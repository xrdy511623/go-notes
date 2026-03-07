package enumerdemo

import (
	"encoding/json"
	"testing"
)

// TestStatusString 验证String()方法（trimprefix去掉了"Status"）
func TestStatusString(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusPending, "Pending"},
		{StatusActive, "Active"},
		{StatusInactive, "Inactive"},
		{StatusDeleted, "Deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status(%d).String() = %q, want %q", int(tt.status), got, tt.want)
			}
		})
	}
}

// TestStatusFromString 验证字符串→枚举值解析
func TestStatusFromString(t *testing.T) {
	tests := []struct {
		input   string
		want    Status
		wantErr bool
	}{
		{"Pending", StatusPending, false},
		{"Active", StatusActive, false},
		{"Unknown", 0, true}, // 无效字符串
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := StatusString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("StatusString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("StatusString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestStatusJSON 验证JSON序列化/反序列化（输出字符串而非数字）
func TestStatusJSON(t *testing.T) {
	type Order struct {
		ID     string `json:"id"`
		Status Status `json:"status"`
	}

	// 序列化：数字→字符串
	order := Order{ID: "ord-1", Status: StatusActive}
	data, err := json.Marshal(order)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	want := `{"id":"ord-1","status":"Active"}`
	if got != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}

	// 反序列化：字符串→枚举
	var decoded Order
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Status != StatusActive {
		t.Errorf("Unmarshal status = %v, want %v", decoded.Status, StatusActive)
	}

	t.Logf("JSON输出: %s（是字符串而非数字，对比stringer的 {\"status\":1}）", got)
}

// TestStatusValues 验证枚举值列表
func TestStatusValues(t *testing.T) {
	values := StatusValues()
	if len(values) != 4 {
		t.Fatalf("StatusValues() len = %d, want 4", len(values))
	}

	expected := []Status{StatusPending, StatusActive, StatusInactive, StatusDeleted}
	for i, v := range values {
		if v != expected[i] {
			t.Errorf("values[%d] = %v, want %v", i, v, expected[i])
		}
	}
}

// TestStatusIsValid 验证有效性校验
func TestStatusIsValid(t *testing.T) {
	if !StatusPending.IsAStatus() {
		t.Error("StatusPending should be valid")
	}
	if Status(99).IsAStatus() {
		t.Error("Status(99) should be invalid")
	}
}
