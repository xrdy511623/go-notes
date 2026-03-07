package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestColorString(t *testing.T) {
	tests := []struct {
		color Color
		want  string
	}{
		{ColorRed, "Red"},
		{ColorGreen, "Green"},
		{ColorBlue, "Blue"},
		{ColorYellow, "Yellow"},
		{Color(99), "Color(99)"}, // 未知值的安全处理
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.color.String()
			if got != tt.want {
				t.Errorf("Color(%d).String() = %q, want %q", int(tt.color), got, tt.want)
			}
		})
	}
}

func TestWeekdayString(t *testing.T) {
	tests := []struct {
		day  Weekday
		want string
	}{
		{Monday, "周一"},
		{Friday, "周五"},
		{Sunday, "周日"},
		{Weekday(0), "Weekday(0)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.day.String()
			if got != tt.want {
				t.Errorf("Weekday(%d).String() = %q, want %q", int(tt.day), got, tt.want)
			}
		})
	}
}

// TestColorInFormat 验证fmt系列函数自动调用String()
func TestColorInFormat(t *testing.T) {
	got := fmt.Sprintf("color is %s", ColorRed)
	want := "color is Red"
	if got != want {
		t.Errorf("Sprintf = %q, want %q", got, want)
	}
}

// TestColorInJSON 演示stringer不自动提供JSON支持
// （需要enumer或手动实现MarshalJSON）
func TestColorInJSON(t *testing.T) {
	type Palette struct {
		Primary Color `json:"primary"`
	}

	p := Palette{Primary: ColorRed}
	data, _ := json.Marshal(p)
	// stringer只生成String()，JSON序列化仍然是数字
	got := string(data)
	want := `{"primary":0}`
	if got != want {
		t.Errorf("JSON = %s, want %s", got, want)
	}
	t.Logf("注意: stringer不生成JSON方法，序列化结果是数字: %s", got)
	t.Log("如需JSON字符串序列化，请使用enumer（见 enumer-demo/）")
}
