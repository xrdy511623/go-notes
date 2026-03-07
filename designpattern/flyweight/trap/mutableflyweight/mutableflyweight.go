package mutableflyweight

import "fmt"

// ❌ 反模式：享元对象包含可变状态
//
// 如果共享的享元是可变的，一个调用方的修改会影响所有引用者：
//
//	icon := factory.Get("star")
//	icon.Scale = 2.0  // ← 所有引用 "star" 的地方都变成了 2.0！
//
// ✅ 正确做法：享元只包含不可变的内在状态（intrinsic），
// 可变的外在状态（extrinsic）由调用方自己管理：
//
//	icon := factory.Get("star")   // 共享的不可变数据
//	render(icon, x, y, scale)     // 外在状态作为参数传入

// MutableIcon 演示了可变享元的危险。
type MutableIcon struct {
	Name  string
	Data  []byte
	Scale float64 // ❌ 可变字段！共享时会被意外修改
}

var shared = &MutableIcon{Name: "star", Data: []byte{0xFF}, Scale: 1.0}

// Bad 展示两个调用方共享同一个可变享元的问题。
func Bad() {
	a := shared
	b := shared

	fmt.Printf("修改前: a.Scale=%.1f b.Scale=%.1f\n", a.Scale, b.Scale)
	a.Scale = 2.0
	fmt.Printf("修改后: a.Scale=%.1f b.Scale=%.1f\n", a.Scale, b.Scale)
	// Output:
	//   修改前: a.Scale=1.0 b.Scale=1.0
	//   修改后: a.Scale=2.0 b.Scale=2.0  ← b 被意外修改！
}
