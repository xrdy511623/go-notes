package simplefactory

import "fmt"

// Scene 标识消息发送渠道。
type Scene int

const (
	SceneDingDing Scene = iota + 1
	SceneWeixin
	SceneFeishu
)

// String 返回 Scene 的可读名称，实现 fmt.Stringer 接口。
func (s Scene) String() string {
	switch s {
	case SceneDingDing:
		return "dingding"
	case SceneWeixin:
		return "weixin"
	case SceneFeishu:
		return "feishu"
	default:
		return fmt.Sprintf("Scene(%d)", int(s))
	}
}

// Scenes 返回所有已定义的 Scene 常量。
func Scenes() []Scene {
	return []Scene{SceneDingDing, SceneWeixin, SceneFeishu}
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
