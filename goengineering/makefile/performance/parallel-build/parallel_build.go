package parallelbuild

/*
性能对比：并行构建 vs 串行构建

Go 编译器天然支持并行编译（-p 参数控制并行数）。
Makefile 也支持 -j 参数控制 target 并行执行。

本实验模拟：
  1. 串行编译多个包：逐个编译，上一个完成再编译下一个
  2. 并行编译多个包：利用多核同时编译多个独立包

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s ./...

关键结论：
  - go build 默认使用 GOMAXPROCS 个并行编译进程
  - 对于多包项目，并行编译速度提升与 CPU 核数正相关
  - Makefile 中可以用 make -j$(nproc) 并行执行多个 target
  - 在 CI 中，合理配置并行度可以显著缩短构建时间
*/

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// CompileUnit 模拟一个编译单元（包）
type CompileUnit struct {
	Name   string
	Source []byte // 模拟源码
}

// NewCompileUnits 创建 n 个模拟编译单元
func NewCompileUnits(n int) []CompileUnit {
	units := make([]CompileUnit, n)
	for i := range units {
		source := make([]byte, 4096) // 4KB 模拟源码
		for j := range source {
			source[j] = byte((i*7 + j*13) % 256)
		}
		units[i] = CompileUnit{
			Name:   fmt.Sprintf("pkg%d", i),
			Source: source,
		}
	}
	return units
}

// compile 模拟编译一个包（CPU 密集操作）
func compile(unit CompileUnit) [32]byte {
	// 模拟编译：多轮哈希运算
	hash := sha256.Sum256(unit.Source)
	for i := 0; i < 100; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// SequentialBuild 串行编译所有包
func SequentialBuild(units []CompileUnit) [][32]byte {
	results := make([][32]byte, len(units))
	for i, unit := range units {
		results[i] = compile(unit)
	}
	return results
}

// ParallelBuild 并行编译所有包
func ParallelBuild(units []CompileUnit, workers int) [][32]byte {
	results := make([][32]byte, len(units))
	var wg sync.WaitGroup

	ch := make(chan int, len(units))
	for i := range units {
		ch <- i
	}
	close(ch)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				results[i] = compile(units[i])
			}
		}()
	}

	wg.Wait()
	return results
}
