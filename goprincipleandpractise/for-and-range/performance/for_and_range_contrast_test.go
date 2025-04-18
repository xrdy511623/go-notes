package performance

import (
	"testing"
)

type Item struct {
	id  int
	val [4096]byte
}

// 当遍历对象是int数组(切片)时，for与range相比几乎没有性能差异，因为遍历的每个元素都是数字，本身占用内存很小
func BenchmarkForIntSlice(b *testing.B) {
	nums := GenerateWithCap(1024 * 1024)
	for i := 0; i < b.N; i++ {
		length := len(nums)
		var tmp int
		for k := 0; k < length; k++ {
			tmp = nums[k]
		}
		_ = tmp
	}
}

func BenchmarkRangeIntSlice(b *testing.B) {
	nums := GenerateWithCap(1024 * 1024)
	for i := 0; i < b.N; i++ {
		var tmp int
		for _, num := range nums {
			tmp = num
		}
		_ = tmp
	}
}

/*
如果换成占用内存较大的结构体，结果却有所不同:
仅遍历下标的情况下，for 和 range 的性能几乎是一样的。
items 的每一个元素的类型是一个结构体类型 Item，Item 由两个字段构成，一个类型是 int，一个是类型是 [4096]byte，也就是说每个
Item 实例需要申请约 4KB 的内存。
在这个例子中，for 的性能大约是 range (同时遍历下标和值) 的 500 倍。

与 for循环不同的是，range 对每个迭代值都创建了一个拷贝。因此如果每次迭代的值内存占用很小的情况下，for 和 range
的性能几乎没有差异，但是如果每个迭代值内存占用很大，例如上面的例子中，每个结构体需要占据 4KB 的内存，这种情况下性能差距
就非常明显了。
*/
func BenchmarkForStruct(b *testing.B) {
	var items [1024]Item
	for i := 0; i < b.N; i++ {
		length := len(items)
		var tmp int
		for k := 0; k < length; k++ {
			tmp = items[k].id
		}
		_ = tmp
	}
}

func BenchmarkRangeIndexStruct(b *testing.B) {
	var items [1024]Item
	for i := 0; i < b.N; i++ {
		var tmp int
		for k := range items {
			tmp = items[k].id
		}
		_ = tmp
	}
}

func BenchmarkRangeStruct(b *testing.B) {
	var items [1024]Item
	for i := 0; i < b.N; i++ {
		var tmp int
		for _, item := range items {
			tmp = item.id
		}
		_ = tmp
	}
}

// 如果切片或数组中的元素是结构体的指针呢？
// 从测试结果来看，切片元素从结构体替换为指针后，for 和 range 的性能几乎是一样的。
// 而且使用指针还有另一个好处，可以直接修改指针对应的结构体的值。
func generateItems(n int) []*Item {
	items := make([]*Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, &Item{id: i})
	}
	return items
}

func BenchmarkForPointer(b *testing.B) {
	items := generateItems(1024)
	for i := 0; i < b.N; i++ {
		length := len(items)
		var tmp int
		for k := 0; k < length; k++ {
			tmp = items[k].id
		}
		_ = tmp
	}
}

func BenchmarkRangePointer(b *testing.B) {
	items := generateItems(1024)
	for i := 0; i < b.N; i++ {
		var tmp int
		for _, item := range items {
			tmp = item.id
		}
		_ = tmp
	}
}

/*
当我们事先不知道切片的长度时，如果for循环遍历时预先计算出切片的长度，而不是每次循环都去计算长度比较下标是否越界，
可以提升性能，在本案例中，性能提升了大约7.3%。
go test -bench=Loop$ -benchmem -count=3 .
goos: darwin
goarch: arm64
pkg: go-notes/for-and-range/performance
BenchmarkNormalLoop-8           1000000000               0.0004530 ns/op               0 B/op          0 allocs/op
BenchmarkNormalLoop-8           1000000000               0.0004086 ns/op               0 B/op          0 allocs/op
BenchmarkNormalLoop-8           1000000000               0.0004117 ns/op               0 B/op          0 allocs/op
BenchmarkEnhanceLoop-8          1000000000               0.0004090 ns/op               0 B/op          0 allocs/op
BenchmarkEnhanceLoop-8          1000000000               0.0003769 ns/op               0 B/op          0 allocs/op
BenchmarkEnhanceLoop-8          1000000000               0.0003943 ns/op               0 B/op          0 allocs/op
PASS
ok      go-notes/for-and-range/performance      0.444s
*/

var s = GenerateWithCap(1000000)

func BenchmarkNormalLoop(b *testing.B) {
	NormalLoop(s)
}

func BenchmarkEnhanceLoop(b *testing.B) {
	EnhanceLoop(s)
}
