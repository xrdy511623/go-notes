package main

import (
	"crypto/md5" //nolint:gosec // 故意演示反例
	"crypto/sha256"
	"fmt"
	"time"
)

/*
陷阱：用快速哈希（MD5/SHA1/SHA256）存储密码

运行：go run .

预期行为：
  MD5、SHA1、SHA256 等通用哈希函数设计目标是"快"，
  现代 GPU 每秒可计算数十亿次 MD5，暴力破解密码轻而易举。

  密码存储必须用专门的慢哈希（key derivation function）：
  - bcrypt：经典选择，自带 salt，cost 可调
  - Argon2id：现代推荐，抗 GPU/ASIC，内存硬化
  - PBKDF2：标准合规场景（NIST 推荐）

  本示例通过计时展示 MD5 vs 迭代 SHA-256 的速度差异，
  说明为什么密码哈希必须"慢"。

  正确做法：使用 golang.org/x/crypto/bcrypt 或 argon2id。
*/

func main() {
	password := []byte("P@ssw0rd123!")

	fmt.Println("=== 错误做法：快速哈希存储密码 ===")

	// MD5 单次
	start := time.Now()
	for i := 0; i < 1_000_000; i++ {
		md5.Sum(password) //nolint:gosec // 故意演示反例
	}
	md5Duration := time.Since(start)
	fmt.Printf("  MD5  x 1,000,000 次: %v (%.0f ns/op)\n", md5Duration, float64(md5Duration.Nanoseconds())/1_000_000)
	fmt.Println("  攻击者用 GPU 每秒可算数十亿次，几小时破解大部分密码")

	// SHA256 单次
	start = time.Now()
	for i := 0; i < 1_000_000; i++ {
		sha256.Sum256(password)
	}
	sha256Duration := time.Since(start)
	fmt.Printf("  SHA256 x 1,000,000 次: %v (%.0f ns/op)\n", sha256Duration, float64(sha256Duration.Nanoseconds())/1_000_000)
	fmt.Println("  SHA256 比 MD5 慢一点，但仍然太快，不适合做密码哈希")

	fmt.Println("\n=== 正确思路：迭代哈希（模拟慢哈希原理） ===")

	// 迭代 SHA-256 模拟 PBKDF2 原理
	iterations := []int{1_000, 10_000, 100_000}
	for _, iter := range iterations {
		start = time.Now()
		h := sha256.Sum256(password)
		for i := 1; i < iter; i++ {
			h = sha256.Sum256(h[:])
		}
		duration := time.Since(start)
		fmt.Printf("  SHA256 x %d 次迭代: %v\n", iter, duration)
	}

	fmt.Println("\n=== 推荐方案（不在此处 import，避免新增依赖） ===")
	fmt.Println("  bcrypt:")
	fmt.Println("    import \"golang.org/x/crypto/bcrypt\"")
	fmt.Println("    hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)")
	fmt.Println("    err = bcrypt.CompareHashAndPassword(hash, password)")
	fmt.Println("    特点：自带 salt，cost 参数控制慢度，默认 cost=10 约 100ms")
	fmt.Println()
	fmt.Println("  argon2id:")
	fmt.Println("    import \"golang.org/x/crypto/argon2\"")
	fmt.Println("    hash := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)")
	fmt.Println("    特点：内存硬化，抗 GPU/ASIC，time=1, memory=64MB, threads=4")

	fmt.Println("\n=== 速度对比总结 ===")
	fmt.Printf("  MD5 单次:         ~%.0f ns\n", float64(md5Duration.Nanoseconds())/1_000_000)
	fmt.Printf("  SHA256 单次:      ~%.0f ns\n", float64(sha256Duration.Nanoseconds())/1_000_000)
	fmt.Println("  bcrypt (cost=10): ~100,000,000 ns (100ms)")
	fmt.Println("  argon2id:         ~200,000,000 ns (200ms)")
	fmt.Println("  慢 10万+ 倍 = 暴力破解从几小时变成几百年")

	fmt.Println("\n总结:")
	fmt.Println("  1. MD5/SHA1/SHA256 太快，不能用于密码存储")
	fmt.Println("  2. 密码哈希必须用专用慢哈希：bcrypt 或 argon2id")
	fmt.Println("  3. 慢哈希自带 salt，每个密码的哈希值不同")
	fmt.Println("  4. 通用哈希适合校验数据完整性，不适合存密码")
	fmt.Println("  5. gosec G401 检测 MD5/SHA1 使用")
}
