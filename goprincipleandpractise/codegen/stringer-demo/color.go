// Package stringerdemo 演示stringer代码生成。
//
// 使用方式：
//
//	go generate ./...
//	go run .
package main

import "fmt"

//go:generate stringer -type=Color -trimprefix=Color

// Color 定义颜色枚举
type Color int

const (
	ColorRed Color = iota
	ColorGreen
	ColorBlue
	ColorYellow
)

//go:generate stringer -type=Weekday -linecomment

// Weekday 定义星期枚举（使用linecomment自定义字符串）
type Weekday int

const (
	Monday    Weekday = iota + 1 // 周一
	Tuesday                      // 周二
	Wednesday                    // 周三
	Thursday                     // 周四
	Friday                       // 周五
	Saturday                     // 周六
	Sunday                       // 周日
)

func main() {
	// Color使用trimprefix，去掉"Color"前缀
	fmt.Println("=== stringer -trimprefix ===")
	fmt.Println(ColorRed)   // Red
	fmt.Println(ColorGreen) // Green
	fmt.Println(ColorBlue)  // Blue
	fmt.Println(Color(99))  // Color(99) — 未知值的安全处理

	// Weekday使用linecomment，输出行尾注释
	fmt.Println("\n=== stringer -linecomment ===")
	fmt.Println(Monday)     // 周一
	fmt.Println(Friday)     // 周五
	fmt.Println(Sunday)     // 周日
	fmt.Println(Weekday(0)) // Weekday(0) — 未定义的值
}
