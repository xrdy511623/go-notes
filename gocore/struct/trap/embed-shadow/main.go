package main

import "fmt"

/*
陷阱：struct 嵌入时的字段/方法遮蔽（shadowing）

运行：go run .

预期行为：
  当外层 struct 定义了与嵌入 struct 同名的字段或方法时，
  外层的定义会"遮蔽"嵌入的定义。访问时优先使用外层的版本。
  这不是覆盖（override），嵌入的原始方法仍然可以通过显式路径访问。

  多个嵌入 struct 有同名字段/方法时，如果外层没有同名定义，
  编译器会报 "ambiguous selector" 错误。
*/

// ---------- 演示1: 方法遮蔽 ----------

type Logger struct{}

func (Logger) Log(msg string) {
	fmt.Printf("  [Logger] %s\n", msg)
}

type Service struct {
	Logger // 嵌入 Logger
}

// Log 遮蔽了 Logger.Log
func (s Service) Log(msg string) {
	fmt.Printf("  [Service] %s\n", msg)
}

// ---------- 演示2: 字段遮蔽 ----------

type Base struct {
	Name string
}

func (b Base) Hello() string {
	return fmt.Sprintf("Hello from Base, name=%s", b.Name)
}

type Derived struct {
	Base
	Name string // 遮蔽了 Base.Name
}

// ---------- 演示3: 多重嵌入歧义 ----------

type A struct{}

func (A) Do() string { return "A.Do" }

type B struct{}

func (B) Do() string { return "B.Do" }

type C struct {
	A
	B
}

// Do 如果 C 不定义自己的 Do()，调用 c.Do() 会编译报错：ambiguous selector
func (C) Do() string { return "C.Do (resolves ambiguity)" }

func main() {
	fmt.Println("=== 演示1: 方法遮蔽 ===")
	s := Service{}
	s.Log("hello")        // 调用 Service.Log（外层遮蔽内层）
	s.Logger.Log("hello") // 显式调用被遮蔽的 Logger.Log
	fmt.Println("  注意: s.Log 调用的是 Service 的方法，不是 Logger 的")

	fmt.Println("\n=== 演示2: 字段遮蔽 ===")
	d := Derived{
		Base: Base{Name: "base-name"},
		Name: "derived-name",
	}
	fmt.Printf("  d.Name = %q（外层字段）\n", d.Name)
	fmt.Printf("  d.Base.Name = %q（被遮蔽的嵌入字段）\n", d.Base.Name)
	fmt.Printf("  d.Hello() = %q\n", d.Hello())
	fmt.Println("  注意: Base.Hello() 使用的是 Base.Name，不是 Derived.Name！")
	fmt.Println("  嵌入方法的 receiver 仍然是 Base，不会自动使用外层字段")

	fmt.Println("\n=== 演示3: 多重嵌入歧义 ===")
	c := C{}
	fmt.Printf("  c.Do() = %q\n", c.Do())
	fmt.Printf("  c.A.Do() = %q\n", c.A.Do())
	fmt.Printf("  c.B.Do() = %q\n", c.B.Do())
	fmt.Println("  如果 C 不定义 Do()，c.Do() 将编译报错: ambiguous selector")

	fmt.Println("\n总结:")
	fmt.Println("  1. 外层同名字段/方法会遮蔽嵌入的字段/方法")
	fmt.Println("  2. 嵌入方法的 receiver 仍然是被嵌入的类型，不会自动绑定外层")
	fmt.Println("  3. 多个嵌入有同名方法时，外层必须显式定义来消除歧义")
}
