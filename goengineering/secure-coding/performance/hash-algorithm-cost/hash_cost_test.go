package hashalgorithmcost

import "testing"

/*
对比不同哈希算法和迭代次数的性能开销。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下参考结果:

	BenchmarkMD5             单次 MD5:           ~40 ns/op
	BenchmarkSHA256          单次 SHA256:         ~50 ns/op
	BenchmarkIterSHA256_1K   SHA256 x 1000:      ~45,000 ns/op (45µs)
	BenchmarkIterSHA256_10K  SHA256 x 10000:     ~450,000 ns/op (450µs)
	BenchmarkIterSHA256_100K SHA256 x 100000:    ~4,500,000 ns/op (4.5ms)

结论:
  单次 MD5/SHA256 仅需几十纳秒，GPU 每秒可算数十亿次。
  迭代 1000 次达到微秒级，10 万次达到毫秒级。
  bcrypt (cost=10) 约 100ms，argon2id 约 200ms —— 比 MD5 慢百万倍以上。
  密码哈希必须"慢"才能抵御暴力破解。
*/

var password = []byte("P@ssw0rd123!")

func BenchmarkMD5(b *testing.B) {
	for b.Loop() {
		MD5Sum(password)
	}
}

func BenchmarkSHA256(b *testing.B) {
	for b.Loop() {
		SHA256Sum(password)
	}
}

func BenchmarkIterSHA256_1K(b *testing.B) {
	for b.Loop() {
		IteratedSHA256(password, 1_000)
	}
}

func BenchmarkIterSHA256_10K(b *testing.B) {
	for b.Loop() {
		IteratedSHA256(password, 10_000)
	}
}

func BenchmarkIterSHA256_100K(b *testing.B) {
	for b.Loop() {
		IteratedSHA256(password, 100_000)
	}
}
