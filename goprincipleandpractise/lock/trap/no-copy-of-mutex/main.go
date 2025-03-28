package main

import (
	"fmt"
	"sync"
	"time"
)

var num int

func addWrong(m sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	for i := 0; i < 1000; i++ {
		num++
		time.Sleep(time.Microsecond)
	}
}

func addRight(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	for i := 0; i < 1000; i++ {
		num++
		time.Sleep(time.Microsecond)
	}
}

func main() {
	var m sync.Mutex
	go addWrong(m)
	go addWrong(m)
	//go addRight(&m)
	//go addRight(&m)
	time.Sleep(time.Second * 2)
	fmt.Println("num = ", num)
}
