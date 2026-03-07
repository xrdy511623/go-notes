package main

import "fmt"

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
		x = 6
	}()
	return x
}

func df3() (x int) {
	defer func() {
		x++ // 修改的就是返回值x
	}()
	return 5
}

/*
第一步，返回值赋值x=5,第二步defer语句执行，此时修改的就是返回值x，x=6；最后return返回的就是6
*/
func df4() (y int) {
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
func df5() (x int) {
	defer func(x int) {
		x++
	}(x)
	return 5
}

/*
第一步，返回值赋值x=5,第二步defer语句执行，此时内部这个匿名函数修改的是自己内部的函数变量x，与外部函数df5的变量x无关，所以最后返回的还是5
所以最后return返回的就是x=5
*/

func df6() {
	var a = 1
	defer fmt.Println(a)
	a = 2
	return
}

func df7() {
	i := 0
	defer func() {
		fmt.Println(i)
	}()
	i++
}

func df8() {
	i := 0
	defer func(i int) {
		fmt.Println(i)
	}(i)
	i++
}

func df9() {
	arr := [3]int{1, 2, 3}
	defer func(array *[3]int) {
		for i := range array {
			fmt.Println(array[i])
		}
	}(&arr)
	arr[0] = 10
}

func df10() {
	defer func() {
		defer func() {
			fmt.Println("B")
		}()
		fmt.Println("A")
	}()
}

func calc(index string, a, b int) int {
	ret := a + b
	fmt.Println(index, a, b, ret)
	return ret
}

func f1() int {
	var i int
	defer func() {
		i++
	}()
	return 1
}

func f2() int {
	var i int
	defer func() {
		i++
	}()
	return i
}

func f3() (ret int) {
	defer func() {
		ret++
	}()
	return 0
}

func f4() {
	i := 0
	defer fmt.Println(i)
	i++
}

func f5() {
	i := 0
	defer func() {
		fmt.Println(i)
	}()
	i++
}

func f6() {
	i := 0
	defer func(i int) {
		fmt.Println(i)
	}(i)
	i++
}

func main() {
	fmt.Println(df1())
	fmt.Println(df2())
	fmt.Println(df3())
	fmt.Println(df4())
	fmt.Println(df5())
	df6()
	df7()
	df8()
	df9()
	df10()
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
	fmt.Println(f1())
	fmt.Println(f2())
	fmt.Println(f3())
	f4()
	f5()
	f6()
}
