package main

import (
	"fmt"
	"time"
)

/*
陷阱：time.Parse 默认使用 UTC 而非本地时区

运行：go run .

预期行为：
  time.Parse 在解析不含时区信息的字符串时，默认使用 UTC。
  这在东八区（UTC+8）等非 UTC 时区的服务器上会导致 8 小时偏差。

  time.ParseInLocation 允许指定默认时区，应在解析无时区字符串时使用。

  正确做法：解析无时区信息的字符串时，始终使用 ParseInLocation
*/

func main() {
	timeStr := "2024-06-15 14:30:00"
	layout := "2006-01-02 15:04:05"

	fmt.Println("=== time.Parse vs time.ParseInLocation ===")
	fmt.Printf("  输入字符串: %q（无时区信息）\n\n", timeStr)

	// time.Parse：默认 UTC
	t1, err := time.Parse(layout, timeStr)
	if err != nil {
		panic(err)
	}
	fmt.Println("--- time.Parse（默认 UTC）---")
	fmt.Printf("  解析结果:  %s\n", t1)
	fmt.Printf("  时区:      %s\n", t1.Location())
	fmt.Printf("  Unix 时间: %d\n", t1.Unix())

	// time.ParseInLocation：使用 Local 时区
	t2, err := time.ParseInLocation(layout, timeStr, time.Local)
	if err != nil {
		panic(err)
	}
	fmt.Println("\n--- time.ParseInLocation（Local 时区）---")
	fmt.Printf("  解析结果:  %s\n", t2)
	fmt.Printf("  时区:      %s\n", t2.Location())
	fmt.Printf("  Unix 时间: %d\n", t2.Unix())

	// 它们代表不同的时刻
	diff := t1.Sub(t2)
	fmt.Printf("\n--- 差异 ---")
	fmt.Printf("\n  时间差: %v\n", diff)
	fmt.Printf("  Equal: %v\n", t1.Equal(t2))

	if diff != 0 {
		fmt.Printf("  ⚠ 偏差 %v！在非 UTC 时区的服务器上会导致业务逻辑错误\n", diff)
	} else {
		fmt.Println("  （当前系统时区为 UTC，所以没有差异）")
	}

	fmt.Println("\n=== 带时区信息的字符串不受影响 ===")
	timeStrWithTZ := "2024-06-15T14:30:00+08:00"
	t3, _ := time.Parse(time.RFC3339, timeStrWithTZ)
	t4, _ := time.ParseInLocation(time.RFC3339, timeStrWithTZ, time.Local)
	fmt.Printf("  Parse:           %s\n", t3)
	fmt.Printf("  ParseInLocation: %s\n", t4)
	fmt.Printf("  Equal: %v（带时区信息时两者结果一致）\n", t3.Equal(t4))

	fmt.Println("\n=== 实际业务场景 ===")
	// 数据库返回的时间字符串通常不含时区信息
	dbTimeStr := "2024-06-15 14:30:00"

	// 错误：用 Parse，数据库时间被当作 UTC
	wrongTime, _ := time.Parse(layout, dbTimeStr)

	// 正确：数据库与服务器在同一时区，用 ParseInLocation
	correctTime, _ := time.ParseInLocation(layout, dbTimeStr, time.Local)

	fmt.Printf("  数据库返回: %q\n", dbTimeStr)
	fmt.Printf("  Parse 结果:           %s (UTC)\n", wrongTime)
	fmt.Printf("  ParseInLocation 结果: %s (Local)\n", correctTime)

	fmt.Println("\n总结:")
	fmt.Println("  1. time.Parse 对无时区字符串默认使用 UTC")
	fmt.Println("  2. 非 UTC 服务器上会导致时间偏差")
	fmt.Println("  3. 解析无时区字符串时，始终用 time.ParseInLocation")
	fmt.Println("  4. 带时区信息的字符串（如 RFC3339）不受影响")
	fmt.Println("  5. 数据库时间字符串特别容易踩坑，务必注意")
}
