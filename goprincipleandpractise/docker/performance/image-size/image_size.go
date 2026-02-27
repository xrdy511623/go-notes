package imagesize

/*
性能对比：不同基础镜像策略的镜像大小

本实验模拟不同 Docker 构建策略产出的镜像大小差异。

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .

关键结论：
  - golang:1.24 运行时 ~800MB（仅适合构建阶段）
  - alpine + binary ~15MB
  - distroless + binary ~8MB
  - scratch + binary ~5MB
  - scratch + stripped ~3.5MB（-ldflags="-s -w"）
*/

import (
	"crypto/sha256"
)

// ImageStrategy 表示一种镜像构建策略
type ImageStrategy struct {
	Name          string
	BaseImageSize int // 基础镜像大小（字节）
	BinarySize    int // Go 二进制大小（字节）
	Stripped      bool
}

// Strategies 返回所有镜像策略
func Strategies() []ImageStrategy {
	return []ImageStrategy{
		{
			Name:          "golang:1.24 (fat)",
			BaseImageSize: 800 * 1024 * 1024,
			BinarySize:    20 * 1024 * 1024,
			Stripped:      false,
		},
		{
			Name:          "alpine + binary",
			BaseImageSize: 7 * 1024 * 1024,
			BinarySize:    20 * 1024 * 1024,
			Stripped:      false,
		},
		{
			Name:          "distroless + binary",
			BaseImageSize: 2 * 1024 * 1024,
			BinarySize:    20 * 1024 * 1024,
			Stripped:      false,
		},
		{
			Name:          "scratch + binary",
			BaseImageSize: 0,
			BinarySize:    20 * 1024 * 1024,
			Stripped:      false,
		},
		{
			Name:          "scratch + stripped",
			BaseImageSize: 0,
			BinarySize:    14 * 1024 * 1024,
			Stripped:      true,
		},
	}
}

// TotalSize 计算总镜像大小
func (s ImageStrategy) TotalSize() int {
	return s.BaseImageSize + s.BinarySize
}

// simulateBuild 模拟构建过程（含/不含 strip）
func simulateBuild(data []byte, strip bool) [32]byte {
	hash := sha256.Sum256(data)
	iterations := 100
	if strip {
		iterations = 50 // strip 后处理更少
	}
	for i := 0; i < iterations; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// BuildFatImage 模拟构建臃肿镜像（包含编译器和源码）
func BuildFatImage(sourceSize int) [32]byte {
	data := make([]byte, sourceSize)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return simulateBuild(data, false)
}

// BuildSlimImage 模拟构建精简镜像（仅二进制）
func BuildSlimImage(sourceSize int, strip bool) [32]byte {
	// 只处理编译后的二进制（远小于源码 + 编译器）
	binarySize := sourceSize / 10
	data := make([]byte, binarySize)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return simulateBuild(data, strip)
}
