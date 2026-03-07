package performance

import (
	"log/slog"
	"testing"
)

/*
对比 log、slog（Text/JSON）、slog（Attr vs key-value）的性能。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

关注指标:
  - ns/op: 每条日志的延迟
  - B/op: 每条日志的内存分配
  - allocs/op: 分配次数

预期结论:
 1. slog比标准log快（避免fmt.Sprintf）
 2. slog JSON和Text性能接近
 3. LogAttrs（强类型）比交替key-value少1-2次分配
 4. 级别过滤（Enabled检查）几乎零开销
*/

// ==================== log vs slog ====================

func BenchmarkStdLog(b *testing.B) {
	l := NewStdLogger()
	for b.Loop() {
		l.Info("request completed method=%s path=%s status=%d bytes=%d",
			"GET", "/api/users", 200, 1024)
	}
}

func BenchmarkSlogText(b *testing.B) {
	l := NewSlogText()
	for b.Loop() {
		l.Info("request completed",
			"method", "GET",
			"path", "/api/users",
			"status", 200,
			"bytes", 1024,
		)
	}
}

func BenchmarkSlogJSON(b *testing.B) {
	l := NewSlogJSON()
	for b.Loop() {
		l.Info("request completed",
			"method", "GET",
			"path", "/api/users",
			"status", 200,
			"bytes", 1024,
		)
	}
}

// ==================== key-value vs Attr ====================

func BenchmarkSlogKeyValue(b *testing.B) {
	l := NewSlogJSON()
	for b.Loop() {
		LogWithKeyValue(l)
	}
}

func BenchmarkSlogAttrs(b *testing.B) {
	l := NewSlogJSON()
	for b.Loop() {
		LogWithAttrs(l)
	}
}

// ==================== 级别过滤开销 ====================

func BenchmarkSlogDisabledLevel(b *testing.B) {
	// Logger级别为Warn，Info日志会被Enabled()前置拦截
	l := NewSlogJSONWithLevel(slog.LevelWarn)
	for b.Loop() {
		l.Info("this will be filtered out",
			"method", "GET",
			"path", "/api/users",
		)
	}
}

// ==================== With预设字段 ====================

func BenchmarkSlogWithFields(b *testing.B) {
	l := NewSlogJSON().With("request_id", "abc-123", "service", "api")
	for b.Loop() {
		l.Info("request completed",
			"status", 200,
			"bytes", 1024,
		)
	}
}

// ==================== 并发写入 ====================

func BenchmarkSlogJSONParallel(b *testing.B) {
	l := NewSlogJSON()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			l.Info("request completed",
				"method", "GET",
				"path", "/api/users",
				"status", 200,
			)
		}
	})
}
