package randcryptovsmath

import "testing"

/*
对比 crypto/rand 与 math/rand 在不同字节长度下的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下参考结果:

	BenchmarkMathRand16       16 字节 math/rand:   ~30 ns/op
	BenchmarkCryptoRand16     16 字节 crypto/rand:  ~90 ns/op  (~3x 慢)
	BenchmarkMathRand32       32 字节 math/rand:   ~55 ns/op
	BenchmarkCryptoRand32     32 字节 crypto/rand:  ~95 ns/op  (~1.7x 慢)
	BenchmarkMathRandToken    token math/rand:     ~120 ns/op
	BenchmarkCryptoRandToken  token crypto/rand:   ~180 ns/op (~1.5x 慢)

结论:
  crypto/rand 比 math/rand 慢约 1.5-3 倍，但绝对开销仅约 60-90 ns，
  对于 token 生成等低频操作完全可接受。安全场景必须使用 crypto/rand。
*/

func BenchmarkMathRand16(b *testing.B) {
	for b.Loop() {
		MathRandBytes(16)
	}
}

func BenchmarkCryptoRand16(b *testing.B) {
	for b.Loop() {
		CryptoRandBytes(16)
	}
}

func BenchmarkMathRand32(b *testing.B) {
	for b.Loop() {
		MathRandBytes(32)
	}
}

func BenchmarkCryptoRand32(b *testing.B) {
	for b.Loop() {
		CryptoRandBytes(32)
	}
}

func BenchmarkMathRandToken(b *testing.B) {
	for b.Loop() {
		MathRandToken(32)
	}
}

func BenchmarkCryptoRandToken(b *testing.B) {
	for b.Loop() {
		CryptoRandToken(32)
	}
}
