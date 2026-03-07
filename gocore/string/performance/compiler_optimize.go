package performance

var (
	compilerBoolSink bool
	compilerStrSink  string
)

// MapLookupByteSlice 使用 string(b) 作为 map key 查找
// Go 编译器优化: m[string(b)] 不会分配内存
func MapLookupByteSlice(m map[string]int, key []byte) (int, bool) {
	v, ok := m[string(key)]
	return v, ok
}

// MapLookupString 使用 string 作为 map key 查找（基准对照）
func MapLookupString(m map[string]int, key string) (int, bool) {
	v, ok := m[key]
	return v, ok
}

// CompareByteSliceToLiteral 使用 string(b) == "literal" 比较
// Go 编译器优化: string(b) == "literal" 不会分配内存
func CompareByteSliceToLiteral(b []byte) bool {
	return string(b) == "hello"
}

// CompareByteSliceToVar 使用 string(b) == variable 比较
// 注意: 与变量比较时，编译器可能也能优化
func CompareByteSliceToVar(b []byte, s string) bool {
	return string(b) == s
}

// ConcatConstant 常量拼接在编译期完成
func ConcatConstant() string {
	return "hello" + ", " + "world"
}

// ConcatVariable 变量拼接在运行时执行
func ConcatVariable(a, b string) string {
	return a + b
}
