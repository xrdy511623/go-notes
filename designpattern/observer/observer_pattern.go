package observer

import (
	"sync"
)

/*
我们常把23种经典的设计模式分为三类：创建型、结构型、行为型。
我们知道，创建型设计模式主要解决“对象的创建”问题，结构型设计模式主要解决“类或对象的组合或组装”问题，
那行为型设计模式主要解决的就是“类或对象之间的交互”问题。

观察者模式，正是在实际的开发中用的比较多的一种行为型设计模式。

观察者模式（Observer Design Pattern）也被称为发布订阅模式（Publish-Subscribe Design Pattern）。
在GoF的《设计模式》一书中，它的定义是这样的：

Define a one-to-many dependency between objects so that when one object changes state,
all its dependents are notified and updated automatically.

翻译成中文就是：在对象之间定义一个一对多的依赖，当一个对象状态改变的时候，所有依赖的对象都会自动收到通知。
一般情况下，被依赖的对象叫作被观察者（Observable），依赖的对象叫作观察者（Observer）。
*/

/*
要求实现Pub(发布)Sub(订阅)模式，要求所有的订阅者都能收到发布者发布的消息，同时订阅者取消订阅后，不再收到消息
*/

type Subscriber struct {
	Channel chan string
	Id      int
}

type Publisher struct {
	Subs  []Subscriber
	Mutex sync.Mutex
}

// NewPublisher 创建一个新的发布者
func NewPublisher() *Publisher {
	// 初始化空的订阅者列表
	return &Publisher{
		Subs: make([]Subscriber, 0),
	}
}

// Subscribe 新增订阅者
func (pub *Publisher) Subscribe(id int, chanSize int) Subscriber {
	pub.Mutex.Lock()
	defer pub.Mutex.Unlock()
	sub := Subscriber{
		Id: id,
		// 为订阅者创建消息通道
		Channel: make(chan string, chanSize),
	}
	// 将订阅者添加到列表
	pub.Subs = append(pub.Subs, sub)
	// 返回订阅者对象
	return sub
}

// Unsubscribe 取消某个订阅者的订阅
func (pub *Publisher) Unsubscribe(id int) {
	pub.Mutex.Lock()
	defer pub.Mutex.Unlock()
	for i, sub := range pub.Subs {
		if sub.Id == id {
			// 从列表中移除订阅者
			pub.Subs = append(pub.Subs[:i], pub.Subs[i+1:]...)
			// 关闭订阅者的通道
			close(sub.Channel)
			break
		}
	}
}

// Publish 发布消息给所有订阅者
func (pub *Publisher) Publish(msg string) {
	pub.Mutex.Lock()
	defer pub.Mutex.Unlock()
	// 向每个订阅者的通道发送消息
	for _, sub := range pub.Subs {
		sub.Channel <- msg
	}
}
