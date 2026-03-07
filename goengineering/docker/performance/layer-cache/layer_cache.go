package layercache

/*
性能对比：Docker 层缓存命中 vs 未命中

Docker 使用分层文件系统，每条指令创建一个层。
如果某一层的输入没有变化，直接使用缓存层，跳过执行。

正确的 COPY 顺序可以大幅提高缓存命中率：

  ✅ 好的顺序（依赖文件很少变化，缓存命中率高）：
  COPY go.mod go.sum ./     ← 很少变化
  RUN go mod download       ← 缓存命中，跳过下载
  COPY . .                  ← 代码变化，只重建这层
  RUN go build .            ← 重新编译

  ❌ 差的顺序（任何变化都破坏缓存）：
  COPY . .                  ← 任何文件变化都破坏这层
  RUN go mod download       ← 缓存 miss，重新下载
  RUN go build .            ← 缓存 miss，全量编译

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .
*/

import (
	"crypto/sha256"
)

// Layer 模拟 Docker 层
type Layer struct {
	Name     string
	InputKey string // 决定缓存命中的关键输入
	Work     int    // 工作量
}

// buildLayer 模拟构建一个层
func buildLayer(layer Layer) [32]byte {
	data := []byte(layer.InputKey)
	hash := sha256.Sum256(data)
	for i := 0; i < layer.Work; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// GoodLayerOrder 正确的层顺序（高缓存命中率）
// 代码变更时只有最后两层需要重建
func GoodLayerOrder(codeChanged bool) [][32]byte {
	layers := []Layer{
		{Name: "COPY go.mod go.sum", InputKey: "gomod-v1.0", Work: 10},
		{Name: "RUN go mod download", InputKey: "download-v1.0", Work: 500},
	}

	results := make([][32]byte, 0, 4)

	// 前两层缓存命中（go.mod 没变）
	for _, l := range layers {
		results = append(results, buildLayer(l))
	}

	// 代码变更导致后续层需要重建
	if codeChanged {
		results = append(results, buildLayer(Layer{
			Name: "COPY . .", InputKey: "code-v2.0", Work: 10,
		}))
		results = append(results, buildLayer(Layer{
			Name: "RUN go build", InputKey: "build-v2.0", Work: 300,
		}))
	} else {
		results = append(results, buildLayer(Layer{
			Name: "COPY . .", InputKey: "code-v1.0", Work: 10,
		}))
		results = append(results, buildLayer(Layer{
			Name: "RUN go build", InputKey: "build-v1.0", Work: 300,
		}))
	}

	return results
}

// BadLayerOrder 错误的层顺序（低缓存命中率）
// 任何代码变更都导致所有层重建
func BadLayerOrder(codeChanged bool) [][32]byte {
	key := "v1.0"
	if codeChanged {
		key = "v2.0"
	}

	layers := []Layer{
		// COPY . . 在最前面，任何变化都破坏缓存
		{Name: "COPY . .", InputKey: "all-" + key, Work: 10},
		{Name: "RUN go mod download", InputKey: "download-" + key, Work: 500},
		{Name: "RUN go build", InputKey: "build-" + key, Work: 300},
	}

	results := make([][32]byte, len(layers))
	for i, l := range layers {
		results[i] = buildLayer(l)
	}
	return results
}
