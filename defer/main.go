package main

import (
	"fmt"
	"time"
)

func df1() int {
	x := 5
	defer func() {
		x++ // 改的是内部变量x，不是返回值
	}()
	return x
}

/*
第一步，返回值赋值=5,第二步defer语句执行，此时修改的是内部变量x，并不是返回值；最后return返回的就是5
*/

func df2() int {
	x := 5
	defer func() {
		x += 1
		//x = x + 1 // 改的是内部变量x，不是返回值
	}()
	return x
}

func df3() int {
	x := 5
	defer func() {
		x = 6
	}()
	return x
}

/*

 */

func df4() (x int) {
	defer func() {
		x++ // 修改的就是返回值x
	}()
	return 5
}

/*
第一步，返回值赋值x=5,第二步defer语句执行，此时修改的就是返回值x，x=6；最后return返回的就是6
*/
func df5() (y int) {
	x := 5
	defer func() {
		x++
	}()
	return x
}

/*
第一步，返回值赋值y=x=5,第二步defer语句执行，此时修改的是x，x=6，由于x,y都是int类型属于值类型，所以修改x的值，
作为x的副本y并不会随之发生变化
所以最后return返回的就是y=5
*/
func df6() (x int) {
	defer func(x int) {
		x++
	}(x)
	return 5
}

/*
第一步，返回值赋值x=5,第二步defer语句执行，此时内部这个匿名函数修改的是自己内部的函数变量x，与外部函数df4的变量x无关，所以最后返回的还是5
所以最后return返回的就是x=5
*/

func calc(index string, a, b int) int {
	ret := a + b
	fmt.Println(index, a, b, ret)
	return ret
}

func main() {
	fmt.Println(df1())
	fmt.Println(df2())
	fmt.Println(df3())
	fmt.Println(df4())
	fmt.Println(df5())
	fmt.Println(df6())
	/*
		Go语言中的defer语句会将其后面跟随的语句进行延迟处理。在defer归属的函数即将返回或退出时，将延迟处理的语句按
		defer定义的逆序进行执行，也就是说，先被defer的语句最后被执行，最后被defer的语句，最先被执行。
	*/
	x := 1
	y := 2
	defer calc("AA", x, calc("A", x, y))
	x = 10
	defer calc("BB", x, calc("B", x, y))
	y = 20
	/*
		上面的代码执行结果是:
		"A", 1, 2, 3
		"B", 10, 2, 12
		"BB", 10, 12, 22
		"AA", 1, 3, 4
	*/
	fmt.Println("当前时间是:")
	fmt.Println(time.Now().Unix())
}
