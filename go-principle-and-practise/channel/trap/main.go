package main

import (
	"fmt"
	"time"
)

type user struct {
	name string
	age  int8
}

var u = user{name: "Ankur", age: 25}
var g = &u

func modifyUser(pu *user) {
	fmt.Println("modifyUser Received Vaule", pu)
	pu.name = "Anand"
}

func printUser(u <-chan *user) {
	time.Sleep(2 * time.Second)
	fmt.Println("printUser goRoutine called", <-u)
}

func main() {

	/*
		输出:
		&{Ankur 25}
		modifyUser Received Vaule &{Ankur Anand 100}
		printUser goRoutine called &{Ankur 25}
		&{Anand 100}

		value-copy.png
		一开始构造一个结构体 u，地址是 0x56420，图中地址上方就是它的内容。接着把 &u 赋值给指针 g，g 的地址是
		0x565bb0，它的内容就是一个地址，指向 u。
		main 程序里，先把 g 发送到 c，根据 copy value 的本质，进入到 chan buf 里的就是 0x56420，它是
		指针 g 的值（不是它指向的内容），所以打印从 channel 接收到的元素时，它就是 &{Ankur 25}。因此，
		这里并不是将指针 g “发送” 到了 channel 里，只是拷贝它的值而已。
	*/
	c := make(chan *user, 5)
	c <- g
	fmt.Println(g)
	// modify g
	g = &user{name: "Ankur Anand", age: 100}
	go printUser(c)
	go modifyUser(g)
	time.Sleep(5 * time.Second)
	fmt.Println(g)
}
