package registerfactory

import (
	"fmt"
	"sync"
)

/*
程序设计的开闭原则
开闭原则是说，一个模块应该对扩展开放，对修改关闭。
简单工厂和策略设计模式虽然大大增强了程序的可读性和可维护性，但是仍然不满足程序设计的开闭原则，因为NewSendMessageService
函数里仍然有switch case逻辑，如果要增加新地发送消息类型，仍然需要修改这部分代码，这显然是不符合开闭原则的。

我们这里使用注册工厂模式，它允许对象自行注册到一个中央注册中心。这样，当需要创建对象时，可以直接从注册中心获取，
而无需在工厂函数中添加条件判断。这种方法使得代码更容易扩展，因为新增类型只需注册即可，无需修改现有工厂逻辑。
*/

// Sender 定义发送消息的接口
type Sender interface {
	Send(message string) error
}

// senderRegistry 存储不同场景对应的工厂函数
var (
	senderRegistry = make(map[int]func() Sender)
	once           sync.Once
)

// RegisterSender 将具体的发送器注册到工厂中
func RegisterSender(scene int, factory func() Sender) {
	once.Do(func() {
		senderRegistry = make(map[int]func() Sender)
	})
	if _, exists := senderRegistry[scene]; exists {
		panic(fmt.Sprintf("Scene %d already registered", scene))
	}
	senderRegistry[scene] = factory
}

// NewSendMessageService 根据场景创建对应的发送器
func NewSendMessageService(scene int) (Sender, error) {
	factory, ok := senderRegistry[scene]
	if !ok {
		return nil, fmt.Errorf("unsupported scene: %d", scene)
	}
	return factory(), nil
}

// DingDing 实现发送消息的具体发送器
type DingDing struct{}

func (d *DingDing) Send(message string) error {
	fmt.Printf("Using DingDing to send message:%v\n", message)
	return nil
}

// Weixin 实现发送消息的具体发送器
type Weixin struct{}

func (w *Weixin) Send(message string) error {
	fmt.Printf("Using Weixin to send message:%v\n", message)
	return nil
}

// Feishu 实现发送消息的具体发送器
type Feishu struct{}

func (w *Feishu) Send(message string) error {
	fmt.Printf("Using Feishu to send message:%v\n", message)
	return nil
}

// 初始化函数，注册所有默认的发送器
func init() {
	RegisterSender(1, func() Sender { return new(DingDing) })
	RegisterSender(2, func() Sender { return new(Weixin) })
	RegisterSender(3, func() Sender { return new(Feishu) })
	// 如果要增加新地发送消息类型，只需要将其注册到senderRegistry中即可
}

// SendMessage 示例函数，使用工厂函数发送消息
func SendMessage(scene int, message string) error {
	sender, err := NewSendMessageService(scene)
	if err != nil {
		return err
	}
	return sender.Send(message)
}
