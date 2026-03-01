package serialization

import (
	"testing"
)

// ── 单对象序列化 ────────────────────────────────────

func BenchmarkMarshalJSON_Single(b *testing.B) {
	user := SampleUser()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = MarshalJSON(user)
	}
}

func BenchmarkMarshalGob_Single(b *testing.B) {
	user := SampleUser()
	b.ReportAllocs()
	for b.Loop() {
		_, _ = MarshalGob(user)
	}
}

// ── 单对象反序列化 ──────────────────────────────────

func BenchmarkUnmarshalJSON_Single(b *testing.B) {
	data, _ := MarshalJSON(SampleUser())
	b.ReportAllocs()
	for b.Loop() {
		var u User
		_ = UnmarshalJSON(data, &u)
	}
}

func BenchmarkUnmarshalGob_Single(b *testing.B) {
	data, _ := MarshalGob(SampleUser())
	b.ReportAllocs()
	for b.Loop() {
		var u User
		_ = UnmarshalGob(data, &u)
	}
}

// ── 列表序列化（100 个对象）────────────────────────

func BenchmarkMarshalJSON_List100(b *testing.B) {
	users := SampleUsers(100)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = MarshalJSON(users)
	}
}

func BenchmarkMarshalGob_List100(b *testing.B) {
	users := SampleUsers(100)
	b.ReportAllocs()
	for b.Loop() {
		_, _ = MarshalGob(users)
	}
}

// ── 列表反序列化（100 个对象）──────────────────────

func BenchmarkUnmarshalJSON_List100(b *testing.B) {
	data, _ := MarshalJSON(SampleUsers(100))
	b.ReportAllocs()
	for b.Loop() {
		var users []User
		_ = UnmarshalJSON(data, &users)
	}
}

func BenchmarkUnmarshalGob_List100(b *testing.B) {
	data, _ := MarshalGob(SampleUsers(100))
	b.ReportAllocs()
	for b.Loop() {
		var users []User
		_ = UnmarshalGob(data, &users)
	}
}

// ── 编码大小对比 ────────────────────────────────────

func TestEncodedSize(t *testing.T) {
	user := SampleUser()
	jsonData, _ := MarshalJSON(user)
	gobData, _ := MarshalGob(user)

	t.Logf("Single user - JSON: %d bytes, Gob: %d bytes", len(jsonData), len(gobData))

	users := SampleUsers(100)
	jsonList, _ := MarshalJSON(users)
	gobList, _ := MarshalGob(users)

	t.Logf("100 users   - JSON: %d bytes, Gob: %d bytes", len(jsonList), len(gobList))
}
