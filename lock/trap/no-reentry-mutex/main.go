package main

import (
	"fmt"
	"sync"
)

/*
出现死锁的原因在于sync包里的互斥锁Mutex(包括读写锁)是不可重入的，重复加锁之前这个锁必须是已经释放
了才可以，本案例中释放锁的操作根据defer语法是后进先出(执行)，所以第二次加锁时，第一次加的锁还未释放，
因为它还在等待第二次的defer操作释放锁，而第二次加锁由于第一次的锁还未释放掉所以无法加锁成功，会一直阻塞，
等待第一次锁的释放，最终导致循环等待，出现死锁的bug。

解决的方案是不使用defer，这样便可顺序加锁和释放锁，但是这个问题的关键在于互斥锁Mutex是不可重入的，
所以不要重复加锁。
*/

func HelloWorld(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	fmt.Println("Hello")
	m.Lock()
	defer m.Unlock()
	fmt.Println("World")
}

func helloWorld(m *sync.Mutex) {
	m.Lock()
	fmt.Println("Hello")
	m.Unlock()
	m.Lock()
	fmt.Println("World")
	m.Unlock()
}

func main() {
	var m sync.Mutex
	//HelloWorld(&m)
	helloWorld(&m)
}
