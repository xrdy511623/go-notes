package main

import (
	"fmt"
	"go-notes/designpattern/observer"
	"go-notes/designpattern/registerfactory"
	"go-notes/designpattern/simplefactory"
	"time"
)

func main() {
	// 使用简单工厂和策略模式来发送消息
	err := simplefactory.SendMessage(3, "简单工厂和策略模式")
	if err != nil {
		fmt.Printf("simplefactory.SendMessage err:%v\n", err)
	}
	e := registerfactory.SendMessage(3, "注册模式")
	if e != nil {
		fmt.Printf("registerfactory.SendMessage err:%v\n", e)
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
