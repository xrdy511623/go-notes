// Package enumerdemo 演示enumer增强版枚举代码生成。
//
// enumer相比stringer，额外生成:
//   - 字符串解析（反序列化）
//   - JSON/Text Marshal/Unmarshal
//   - 枚举值列表
//   - 有效性校验
//
// 使用方式：
//
//	go install github.com/dmarkham/enumer@latest
//	go generate ./...
//	go test -v .
package enumerdemo

//go:generate enumer -type=Status -json -text -trimprefix=Status -output=status_enumer.go

// Status 订单状态枚举
type Status int

const (
	StatusPending  Status = iota // 待处理
	StatusActive                 // 活跃
	StatusInactive               // 不活跃
	StatusDeleted                // 已删除
)
