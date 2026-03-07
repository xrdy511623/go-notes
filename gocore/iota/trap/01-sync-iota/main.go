package main

import "fmt"

const (
	mutexLocked = 1 << iota
	mutexWoken
	mutexStarving
	mutexWaiterShift = iota
)

func main() {
	fmt.Println(mutexLocked, mutexWoken, mutexStarving, mutexWaiterShift)
}
