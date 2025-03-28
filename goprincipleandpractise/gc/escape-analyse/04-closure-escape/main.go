package main

import "fmt"

/*
闭包的定义特性
函数内部定义函数：
一个函数定义在另一个函数内部，也就是存在函数的嵌套。
捕获外部变量：
内部函数引用外部函数的变量，即使外部函数已经返回。
延迟求值：
捕获的变量在闭包中是引用，而不是值的拷贝。
外部函数最后返回了内部函数，内部函数就是闭包
即使外层函数已经执行完毕，内部函数仍然可以访问并修改其引用的外层函数变量。
*/

func Fib() func() int {
	a, b := 0, 1
	return func() int {
		a, b = b, a+b
		return a
	}
}

func main() {
	f := Fib()
	for i := 0; i < 10; i++ {
		fmt.Printf("Fib:%d\n", f())
	}
}
