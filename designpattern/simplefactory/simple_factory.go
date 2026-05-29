package simplefactory

//go:generate stringer -type=Scene -linecomment

import "fmt"

// Scene 标识消息发送渠道。
type Scene int

const (
	SceneDingDing Scene = iota + 1 // dingding
	SceneWeixin                    // weixin
	SceneFeishu                    // feishu
)

// _maxScene 标记枚举上界，供 Scenes() 自动枚举使用。
// 定义为无类型常量，stringer 不会将其纳入 String() 生成。
const _maxScene = SceneFeishu + 1

// Scenes 返回所有已定义的 Scene 常量。
// 实现依赖 _maxScene 哨兵自动枚举，新增常量无需修改本函数。
func Scenes() []Scene {
	out := make([]Scene, 0, int(_maxScene)-1)
	for i := Scene(1); i < _maxScene; i++ {
		out = append(out, i)
	}
	return out
}

// Sender 是消息发送的统一接口。
type Sender interface {
	Send(message string) error
}

// --- 具体实现 ---

type dingDingSender struct{}

func (d *dingDingSender) Send(message string) error {
	fmt.Printf("Using DingDing to send message: %v\n", message)
	return nil
}

type weixinSender struct{}

func (w *weixinSender) Send(message string) error {
	fmt.Printf("Using Weixin to send message: %v\n", message)
	return nil
}

type feishuSender struct{}

func (f *feishuSender) Send(message string) error {
	fmt.Printf("Using Feishu to send message: %v\n", message)
	return nil
}

// NewSender 是简单工厂函数：根据 scene 参数决定创建哪种 Sender。
// 未知的 scene 返回错误而非 nil，避免调用方 nil-pointer panic。
func NewSender(scene Scene) (Sender, error) {
	switch scene {
	case SceneDingDing:
		return &dingDingSender{}, nil
	case SceneWeixin:
		return &weixinSender{}, nil
	case SceneFeishu:
		return &feishuSender{}, nil
	default:
		return nil, fmt.Errorf("simplefactory: unknown scene %v", scene)
	}
}

// MustNewSender 与 NewSender 相同，但在失败时 panic。
// 适用于程序启动阶段，scene 已知且不应出错的场景。
// 设计参考：regexp.MustCompile、template.Must。
func MustNewSender(scene Scene) Sender {
	s, err := NewSender(scene)
	if err != nil {
		panic(err)
	}
	return s
}
