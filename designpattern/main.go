package main

import (
	"fmt"
	"time"

	"go-notes/designpattern/observer"
	_ "go-notes/designpattern/registerfactory/dingding"
	_ "go-notes/designpattern/registerfactory/feishu"
	"go-notes/designpattern/registerfactory/sender"
	_ "go-notes/designpattern/registerfactory/weixin"
	"go-notes/designpattern/simplefactory"
)

func main() {
	// 使用简单工厂和策略模式来发送消息
	err := simplefactory.SendMessage(3, "简单工厂和策略模式")
	if err != nil {
		fmt.Printf("simplefactory.SendMessage err:%v\n", err)
	}
	s, _ := sender.Get("dingding")
	if err := s.Send("Hello"); err != nil {
		fmt.Printf("registerfactory.SendMessage err:%v\n", err)
	}
	// 使用观察者模式
	pub := observer.NewPublisher()
	sub1 := pub.Subscribe(1, 100)
	sub2 := pub.Subscribe(2, 100)

	go func() {
		for msg := range sub1.Channel {
			fmt.Printf("订阅者%d接收到消息%v\n", sub1.Id, msg)
		}
	}()

	go func() {
		for msg := range sub2.Channel {
			fmt.Printf("订阅者%d接收到消息%v\n", sub2.Id, msg)
		}
	}()

	pub.Publish("hello world")
	time.Sleep(time.Second)
	pub.Unsubscribe(1)
	pub.Publish("hello again")
	time.Sleep(time.Second)
}
