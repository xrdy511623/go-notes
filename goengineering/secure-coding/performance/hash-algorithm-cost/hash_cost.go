package hashalgorithmcost

import (
	"crypto/md5" //nolint:gosec // 基准测试对比用
	"crypto/sha256"
)

// MD5Sum 计算 MD5 哈希（不安全，仅供对比）
func MD5Sum(data []byte) [16]byte {
	return md5.Sum(data) //nolint:gosec // 基准测试对比用
}

// SHA256Sum 计算 SHA256 哈希
func SHA256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// IteratedSHA256 迭代 SHA-256（模拟 PBKDF2 原理）
func IteratedSHA256(data []byte, iterations int) [32]byte {
	h := sha256.Sum256(data)
	for i := 1; i < iterations; i++ {
		h = sha256.Sum256(h[:])
	}
	return h
}
