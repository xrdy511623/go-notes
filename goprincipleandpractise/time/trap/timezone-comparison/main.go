package main

import (
	"fmt"
	"time"
)

/*
陷阱：不同时区的 time.Time 使用 == 比较

运行：go run .

预期行为：
  time.Time 的 == 运算符比较所有字段，包括 Location 指针。
  同一个时刻在不同时区表示时，== 返回 false。
  必须使用 time.Equal() 方法进行语义正确的比较。

  正确做法：始终使用 t1.Equal(t2) 而非 t1 == t2
*/

func main() {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 同一时刻，不同时区 ===")

	// 创建一个 UTC 时间
	utcTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	shanghaiTime := utcTime.In(shanghai)
	newYorkTime := utcTime.In(newYork)

	fmt.Printf("  UTC:      %s\n", utcTime)
	fmt.Printf("  Shanghai: %s\n", shanghaiTime)
	fmt.Printf("  New York: %s\n", newYorkTime)

	// 错误：使用 ==
	fmt.Println("\n--- 使用 == 比较（错误）---")
	fmt.Printf("  UTC == Shanghai:  %v  ← 同一时刻却返回 false！\n", utcTime == shanghaiTime)
	fmt.Printf("  UTC == New York:  %v\n", utcTime == newYorkTime)

	// 正确：使用 Equal
	fmt.Println("\n--- 使用 Equal 比较（正确）---")
	fmt.Printf("  UTC.Equal(Shanghai):  %v  ← 正确识别为同一时刻\n", utcTime.Equal(shanghaiTime))
	fmt.Printf("  UTC.Equal(New York):  %v\n", utcTime.Equal(newYorkTime))

	fmt.Println("\n=== Unix 时间戳始终相等 ===")
	fmt.Printf("  UTC Unix:      %d\n", utcTime.Unix())
	fmt.Printf("  Shanghai Unix: %d\n", shanghaiTime.Unix())
	fmt.Printf("  New York Unix: %d\n", newYorkTime.Unix())
	fmt.Printf("  Unix 相等: %v\n", utcTime.Unix() == shanghaiTime.Unix())

	fmt.Println("\n=== 另一个常见错误：map 中用 time.Time 做 key ===")
	m := make(map[time.Time]string)
	m[utcTime] = "UTC"
	m[shanghaiTime] = "Shanghai"
	fmt.Printf("  map 长度: %d（期望 1，实际 %d）\n", 1, len(m))
	fmt.Println("  因为 == 认为它们不同，map 中有两个 key！")

	fmt.Println("\n=== 正确做法：统一时区或使用 Unix 时间戳做 key ===")
	m2 := make(map[int64]string)
	m2[utcTime.Unix()] = "UTC"
	m2[shanghaiTime.Unix()] = "Shanghai" // 覆盖，因为 Unix 时间戳相同
	fmt.Printf("  map 长度: %d（正确）\n", len(m2))

	fmt.Println("\n总结:")
	fmt.Println("  1. 比较 time.Time 始终用 .Equal()，不要用 ==")
	fmt.Println("  2. == 比较 Location 指针，同一时刻不同时区返回 false")
	fmt.Println("  3. 避免用 time.Time 做 map key（除非已统一时区）")
	fmt.Println("  4. Before/After 方法也是语义正确的，可安全使用")
}
