package main

import (
	"fmt"
	"go-notes/design-pattern/registerfactory"
	"go-notes/design-pattern/simplefactory"
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
}
