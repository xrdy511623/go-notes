package main

import (
	"fmt"
	"sync"
)

func main() {
	m1 := make(map[int]int, 1000)
	//rw := sync.RWMutex{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	for i := 1; i <= 1000; i++ {
		m1[i] = i + 1
	}

	go func() {
		defer wg.Done()
		for k := range m1 {
			//rw.Lock()
			m1[k]++
			//rw.Unlock()
		}
	}()

	go func() {
		defer wg.Done()
		for k, v := range m1 {
			//rw.RLock()
			fmt.Printf("k=%d,v=%d\n", k, v)
			//rw.RUnlock()
		}
	}()

	wg.Wait()
	fmt.Println("over")
}
