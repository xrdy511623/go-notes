package main

import "fmt"

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
