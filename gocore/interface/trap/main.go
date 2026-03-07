package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ============================================================
// 陷阱1：nil值接口 vs nil接口
// 接口值在且仅在 类型和值都为nil 时才等于nil。
// 一旦接口持有了具体类型信息（即使值是nil），它就不再等于nil。
// ============================================================

type MyError struct {
	Msg string
}

func (e *MyError) Error() string { return e.Msg }

func trapNilInterfaceVsNilValue() {
	fmt.Println("=== 陷阱1：nil值接口 vs nil接口 ===")

	// 情况1：nil接口——类型和值都是nil
	var err error
	fmt.Println("nil接口 == nil:", err == nil) // true

	// 情况2：非nil接口持有nil值——类型不是nil
	var myErr *MyError                           // myErr是一个nil指针
	err = myErr                                  // err的类型是*MyError，值是nil
	fmt.Println("持有nil值的接口 == nil:", err == nil) // false！
	fmt.Printf("类型: %T, 值: %v\n\n", err, err)
}

// ============================================================
// 陷阱2：error返回值陷阱
// 函数内部使用具体错误类型，即使值为nil，返回给error接口后也不等于nil。
// 这是陷阱1在实际工程中最常见的表现形式。
// ============================================================

// 危险写法：返回的error接口永远不是nil
func doSomethingWrong() error {
	var err *MyError // nil指针
	// ... 某些逻辑，假设没有错误 ...
	return err // 返回的error接口持有*MyError类型信息，不是nil！
}

// 安全写法：显式返回nil
func doSomethingRight() error {
	var err *MyError
	// ... 某些逻辑，假设没有错误 ...
	if err != nil {
		return err
	}
	return nil // 显式返回nil接口
}

func trapErrorReturn() {
	fmt.Println("=== 陷阱2：error返回值陷阱 ===")

	err1 := doSomethingWrong()
	fmt.Println("危险写法 err == nil:", err1 == nil) // false！

	err2 := doSomethingRight()
	fmt.Println("安全写法 err == nil:", err2 == nil) // true
	fmt.Println()
}

// ============================================================
// 陷阱3：指针接收者赋值陷阱
// 如果方法使用指针接收者实现接口，只有指针能赋值给接口，值不行。
// 编译错误：Cat does not implement Speaker (method Speak has pointer receiver)
// ============================================================

type Speaker interface {
	Speak() string
}

type Dog struct{ Name string }

func (d Dog) Speak() string { return d.Name + ": Woof!" } // 值接收者

type Cat struct{ Name string }

func (c *Cat) Speak() string { return c.Name + ": Meow!" } // 指针接收者

func trapPointerReceiver() {
	fmt.Println("=== 陷阱3：指针接收者赋值陷阱 ===")

	// Dog用值接收者实现，值和指针都可以
	var s1 Speaker = Dog{Name: "Buddy"}
	var s2 Speaker = &Dog{Name: "Buddy"}
	fmt.Println("Dog值:", s1.Speak())
	fmt.Println("Dog指针:", s2.Speak())

	// Cat用指针接收者实现，只有指针可以
	// var s3 Speaker = Cat{Name: "Kitty"}  // 编译错误！取消注释可验证
	var s4 Speaker = &Cat{Name: "Kitty"} // OK
	fmt.Println("Cat指针:", s4.Speak())

	fmt.Println("提示: 取消注释 var s3 Speaker = Cat{} 可看到编译错误")
	fmt.Println()
}

// ============================================================
// 陷阱4：接口比较陷阱
// 两个接口值可以用==比较，但如果底层类型不可比较（slice、map、func），
// 运行时会panic。
// ============================================================

func trapInterfaceCompare() {
	fmt.Println("=== 陷阱4：接口比较陷阱 ===")

	// 可比较类型：正常工作
	var a, b interface{}
	a = 42
	b = 42
	fmt.Println("int比较:", a == b) // true

	a = "hello"
	b = "hello"
	fmt.Println("string比较:", a == b) // true

	// 不可比较类型：panic！
	a = []int{1, 2, 3}
	b = []int{1, 2, 3}
	fmt.Println("slice已赋值给接口，准备比较...")

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("捕获panic: %v\n", r)
			}
		}()
		_ = a == b // panic: comparing uncomparable type []int
	}()

	// 同样会panic的类型
	fmt.Println("注意: map、func、包含slice的struct 也不可比较")
	fmt.Println()
}

// ============================================================
// 陷阱5：JSON反序列化的float64陷阱
// encoding/json 将JSON数字反序列化到 interface{} 时统一使用 float64。
// 直接断言为int会panic。
// ============================================================

func trapJSONFloat64() {
	fmt.Println("=== 陷阱5：JSON反序列化的float64陷阱 ===")

	var data interface{}
	_ = json.Unmarshal([]byte(`{"age": 25, "score": 99.5}`), &data)

	m := data.(map[string]interface{})
	age := m["age"]
	fmt.Printf("age的类型: %T, 值: %v\n", age, age) // float64, 25

	// 错误：直接断言为int
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("断言int失败: %v\n", r)
			}
		}()
		_ = age.(int) // panic!
	}()

	// 正确：先断言为float64再转换
	n := int(age.(float64))
	fmt.Println("正确做法 int(age.(float64)):", n)

	// 更好的方式：使用json.Decoder + UseNumber
	dec := json.NewDecoder(strings.NewReader(`{"age": 25}`))
	dec.UseNumber()
	var data2 interface{}
	_ = dec.Decode(&data2)
	m2 := data2.(map[string]interface{})
	num := m2["age"].(json.Number)
	intVal, _ := num.Int64()
	fmt.Printf("UseNumber方式: 类型=%T, 值=%d\n", num, intVal)
	fmt.Println()
}

func main() {
	trapNilInterfaceVsNilValue()
	trapErrorReturn()
	trapPointerReceiver()
	trapInterfaceCompare()
	trapJSONFloat64()
}
