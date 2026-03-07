package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	mathrand "math/rand/v2" //nolint:gosec // 故意演示反例
)

/*
陷阱：用 math/rand 生成安全令牌

运行：go run .

预期行为：
  math/rand 使用伪随机数生成器（PRNG），输出可预测。
  即使用 time.Now().UnixNano() 做种子，攻击者只需猜测纳秒级时间戳
  就能重现整个随机序列，从而伪造 token。

  crypto/rand 读取操作系统熵源（Linux: /dev/urandom, macOS: getentropy），
  生成密码学安全的随机数（CSPRNG），不可预测。

  正确做法：任何涉及安全的随机数（token、密钥、nonce）必须用 crypto/rand。
*/

func main() {
	fmt.Println("=== 错误做法：math/rand 生成 token ===")
	fmt.Println("  math/rand/v2 默认使用自动种子，但仍是 PRNG，输出可预测")
	for i := 0; i < 3; i++ {
		token := weakToken(16)
		fmt.Printf("  token %d: %s\n", i+1, token)
	}
	fmt.Println("  问题：攻击者知道算法和种子后可重现全部输出")

	fmt.Println("\n=== 正确做法：crypto/rand 生成 token ===")
	for i := 0; i < 3; i++ {
		token, err := secureToken(16)
		if err != nil {
			fmt.Printf("  生成失败: %v\n", err)
			continue
		}
		fmt.Printf("  token %d: %s\n", i+1, token)
	}
	fmt.Println("  crypto/rand 读取 OS 熵源，输出不可预测")

	fmt.Println("\n=== 对比：相同种子下 math/rand 完全可复现 ===")
	src := mathrand.NewPCG(42, 0) //nolint:gosec // 故意演示反例
	r1 := mathrand.New(src)
	src2 := mathrand.NewPCG(42, 0) //nolint:gosec // 故意演示反例
	r2 := mathrand.New(src2)
	for i := 0; i < 3; i++ {
		v1, v2 := r1.Int64(), r2.Int64()
		fmt.Printf("  r1=%d, r2=%d, 相同=%v\n", v1, v2, v1 == v2)
	}

	fmt.Println("\n=== crypto/rand 生成安全随机整数 ===")
	for i := 0; i < 3; i++ {
		n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
		fmt.Printf("  安全随机数: %d\n", n)
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. math/rand 是 PRNG，输出可预测，绝不能用于安全场景")
	fmt.Println("  2. crypto/rand 是 CSPRNG，读取 OS 熵源，不可预测")
	fmt.Println("  3. token、密钥、nonce、salt 等必须用 crypto/rand")
	fmt.Println("  4. math/rand 适合模拟、游戏、测试等非安全场景")
}

// weakToken 用 math/rand 生成 hex token（不安全）
func weakToken(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(mathrand.IntN(256)) //nolint:gosec // 故意演示反例
	}
	return hex.EncodeToString(b)
}

// secureToken 用 crypto/rand 生成 hex token（安全）
func secureToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("crypto/rand read failed: %w", err)
	}
	return hex.EncodeToString(b), nil
}
