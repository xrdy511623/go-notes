package simplefactory

import (
	"fmt"
)

/*
简单工厂方法模式的实质是由一个工厂方法根据传入的参数，动态决定应该创建哪一个结构体（这些结构体实现了同一个接口）的实例。
*/

type Sender interface {
	Send(message string) error
}

type DingDing struct{}

func (d *DingDing) Send(message string) error {
	fmt.Printf("Using DingDing to send message:%v\n", message)
	return nil
}

type Weixin struct{}

func (w *Weixin) Send(message string) error {
	fmt.Printf("Using Weixin to send message:%v\n", message)
	return nil
}

type Feishu struct{}

func (w *Feishu) Send(message string) error {
	fmt.Printf("Using Feishu to send message:%v\n", message)
	return nil
}

// NewSendMessageService 简单工厂方法，根据传入参数创建同类对象
func NewSendMessageService(scene int) Sender {
	var impl Sender
	switch scene {
	case 1:
		// 使用钉钉发送消息
		impl = new(DingDing)
	case 2:
		// 使用微信发送消息
		impl = new(Weixin)
	case 3:
		// 使用飞书发送消息
		impl = new(Feishu)
	default:
		return nil
	}
	return impl
}

func SendMessage(scene int, message string) error {
	// 调用简单工厂方法创建对象
	impl := NewSendMessageService(scene)
	return impl.Send(message)
}
