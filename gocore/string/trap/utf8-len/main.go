package main

import (
	"fmt"
	"unicode/utf8"
)

/*
陷阱：len() 返回字节数而非字符数，以及错误的字符串反转

运行：go run .

预期输出：
  len() 返回的是字节数，不是字符数。
  对于包含多字节 UTF-8 字符的字符串，len() 和 RuneCountInString() 结果不同。
  按字节反转 UTF-8 字符串会得到乱码。
*/

func main() {
	s := "Hello, 世界！"

	fmt.Println("=== 陷阱1: len() 返回字节数 ===")
	fmt.Printf("字符串: %q\n", s)
	fmt.Printf("len() = %d (字节数)\n", len(s))
	fmt.Printf("utf8.RuneCountInString() = %d (字符数)\n", utf8.RuneCountInString(s))

	fmt.Println("\n=== 陷阱2: for i vs for range ===")
	fmt.Println("for i (按字节遍历):")
	for i := 0; i < len(s); i++ {
		fmt.Printf("  s[%d] = 0x%02x\n", i, s[i])
	}

	fmt.Println("for range (按 rune 遍历):")
	for i, r := range s {
		fmt.Printf("  index=%d rune=%c (U+%04X)\n", i, r, r)
	}

	fmt.Println("\n=== 陷阱3: 错误的字符串反转 ===")
	fmt.Printf("原始字符串: %s\n", s)
	fmt.Printf("按字节反转（错误）: %s\n", reverseByteWrong(s))
	fmt.Printf("按 rune 反转（正确）: %s\n", reverseRuneCorrect(s))
}

// reverseByteWrong 按字节反转，对于多字节 UTF-8 字符会产生乱码
func reverseByteWrong(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// reverseRuneCorrect 按 rune 反转，正确处理多字节字符
func reverseRuneCorrect(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
