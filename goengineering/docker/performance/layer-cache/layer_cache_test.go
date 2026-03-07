package layercache

import "testing"

/*
基准测试：层缓存命中 vs 未命中

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果：
  BenchmarkGoodLayerOrder_CodeChanged-8    xxx    yyy ns/op
  BenchmarkBadLayerOrder_CodeChanged-8     xxx    yyy ns/op

  好的层顺序：代码变更只需重建 COPY + build 层（~310 工作量）
  差的层顺序：代码变更需重建所有层（~810 工作量）

  差异来源：go mod download（500 工作量）是否被缓存。
*/

func BenchmarkGoodLayerOrder_NoChange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GoodLayerOrder(false)
	}
}

func BenchmarkGoodLayerOrder_CodeChanged(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GoodLayerOrder(true)
	}
}

func BenchmarkBadLayerOrder_NoChange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BadLayerOrder(false)
	}
}

func BenchmarkBadLayerOrder_CodeChanged(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BadLayerOrder(true)
	}
}
