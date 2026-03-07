package performance

// Adder 定义一个简单的接口用于性能测试
type Adder interface {
	Add(a, b int) int
}

// DirectAdder 直接实现加法
type DirectAdder struct{}

func (d DirectAdder) Add(a, b int) int { return a + b }

// Describer 用于 type switch 测试
type Describer interface {
	Describe() string
}

type IntType struct{ V int }

func (t IntType) Describe() string { return "int" }

type StrType struct{ V string }

func (t StrType) Describe() string { return "string" }

type FloatType struct{ V float64 }

func (t FloatType) Describe() string { return "float" }
