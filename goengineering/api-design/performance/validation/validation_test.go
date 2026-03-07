package validation

import "testing"

var validReq = UserRequest{
	Name:  "Alice",
	Email: "alice@example.com",
	Age:   30,
}

// BenchmarkValidateManual 手动校验基准。
func BenchmarkValidateManual(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		ValidateManual(validReq)
	}
}

// BenchmarkValidateReflect 反射校验基准。
func BenchmarkValidateReflect(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		ValidateReflect(validReq)
	}
}

// TestValidatorsAgree 确保两种校验方式对合法输入结果一致。
func TestValidatorsAgree(t *testing.T) {
	tests := []struct {
		name    string
		req     UserRequest
		wantErr bool
	}{
		{"valid", validReq, false},
		{"missing name", UserRequest{Email: "a@b.com", Age: 1}, true},
		{"bad email", UserRequest{Name: "Bob", Email: "bad", Age: 1}, true},
		{"age too high", UserRequest{Name: "Old", Email: "o@e.com", Age: 200}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manual := ValidateManual(tt.req)
			reflect := ValidateReflect(tt.req)
			manualHasErr := len(manual) > 0
			reflectHasErr := len(reflect) > 0

			if manualHasErr != tt.wantErr {
				t.Errorf("manual: hasErr=%v, want=%v, errs=%v", manualHasErr, tt.wantErr, manual)
			}
			if reflectHasErr != tt.wantErr {
				t.Errorf("reflect: hasErr=%v, want=%v, errs=%v", reflectHasErr, tt.wantErr, reflect)
			}
		})
	}
}
