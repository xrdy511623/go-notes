package performance

import (
	"fmt"
	"strings"
	"testing"
)

/*
测试结果参考images目录下的string-concat-contrast.png
从基准测试的结果来看，使用 + 和 fmt.Sprintf 的效率是最低的，和其余的方式相比，性能相差约 1000 倍，而且消耗了
超过 1000 倍的内存。当然 fmt.Sprintf 通常是用来格式化字符串的，一般不会用来拼接字符串。
strings.Builder、bytes.Buffer 和 []byte 的性能差距不大，而且消耗的内存也十分接近，性能最好且消耗内存最小的
是preByteConcat，这种方式预分配了内存，在字符串拼接的过程中，不需要进行字符串的拷贝，也不需要分配新的内存，
因此性能最好，且内存消耗最小。

综合易用性和性能，一般推荐使用 strings.Builder 来拼接字符串。
这是Go官方对 strings.Builder 的解释：
A Builder is used to efficiently build a string using Write methods. It minimizes memory copying.

string.Builder 也提供了预分配内存的方式 Grow
使用了 Grow 优化后的版本的 benchmark 结果对比：

与预分配内存的 []byte 相比，因为省去了 []byte 和字符串(string) 之间的转换，内存分配次数还减少了 1 次，
内存消耗减半。
*/

func BenchmarkPlusConcat(b *testing.B)       { benchmarkConcat(b, PlusConcat) }
func BenchmarkSprintfConcat(b *testing.B)    { benchmarkConcat(b, SprintfConcat) }
func BenchmarkBuilderConcat(b *testing.B)    { benchmarkConcat(b, BuilderConcat) }
func BenchmarkBufferConcat(b *testing.B)     { benchmarkConcat(b, BufferConcat) }
func BenchmarkByteConcat(b *testing.B)       { benchmarkConcat(b, ByteConcat) }
func BenchmarkPreByteConcat(b *testing.B)    { benchmarkConcat(b, PreByteConcat) }
func BenchmarkPreBuilderConcat(b *testing.B) { benchmarkConcat(b, PreBuilderConcat) }

func TestBuilderConcat(t *testing.T) {
	var str = RandomString(10)
	var builder strings.Builder
	capacity := 0
	for i := 0; i < 10000; i++ {
		if builder.Cap() != capacity {
			fmt.Print(builder.Cap(), " ")
			capacity = builder.Cap()
		}
		builder.WriteString(str)
	}
}
