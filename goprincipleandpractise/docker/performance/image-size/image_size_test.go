package imagesize

import (
	"fmt"
	"testing"
)

/*
基准测试：不同镜像策略的构建效率

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果：
  BenchmarkBuildFatImage-8      xxx    yyy ns/op    zzz B/op
  BenchmarkBuildSlimImage/no_strip-8   xxx    yyy ns/op
  BenchmarkBuildSlimImage/stripped-8    xxx    yyy ns/op

  精简镜像不仅体积小，构建过程也更快
  （因为最终阶段只 COPY 一个小文件）。
*/

func BenchmarkBuildFatImage(b *testing.B) {
	sourceSize := 1024 * 100 // 100KB 模拟源码
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BuildFatImage(sourceSize)
	}
}

func BenchmarkBuildSlimImage(b *testing.B) {
	sourceSize := 1024 * 100

	cases := []struct {
		name  string
		strip bool
	}{
		{"no_strip", false},
		{"stripped", true},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				BuildSlimImage(sourceSize, tc.strip)
			}
		})
	}
}

func TestImageSizes(t *testing.T) {
	strategies := Strategies()
	for _, s := range strategies {
		totalMB := float64(s.TotalSize()) / (1024 * 1024)
		t.Logf("%-30s %.1f MB", s.Name, totalMB)
	}
}

func ExampleStrategies() {
	for _, s := range Strategies() {
		totalMB := float64(s.TotalSize()) / (1024 * 1024)
		fmt.Printf("%s: %.1f MB\n", s.Name, totalMB)
	}
	// Output:
	// golang:1.24 (fat): 820.0 MB
	// alpine + binary: 27.0 MB
	// distroless + binary: 22.0 MB
	// scratch + binary: 20.0 MB
	// scratch + stripped: 14.0 MB
}
