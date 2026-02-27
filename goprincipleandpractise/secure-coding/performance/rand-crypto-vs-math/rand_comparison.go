package randcryptovsmath

import (
	"crypto/rand"
	"encoding/hex"
	mathrand "math/rand/v2"
)

// MathRandBytes 使用 math/rand 生成 n 字节（不安全）
func MathRandBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(mathrand.IntN(256)) //nolint:gosec // 基准测试对比用
	}
	return b
}

// CryptoRandBytes 使用 crypto/rand 生成 n 字节（安全）
func CryptoRandBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// MathRandToken 使用 math/rand 生成 hex token（不安全）
func MathRandToken(n int) string {
	return hex.EncodeToString(MathRandBytes(n))
}

// CryptoRandToken 使用 crypto/rand 生成 hex token（安全）
func CryptoRandToken(n int) string {
	return hex.EncodeToString(CryptoRandBytes(n))
}
