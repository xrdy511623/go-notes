package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/syncx"
)

func main() {
	round := 100
	wg := sync.WaitGroup{}
	singleFlight := syncx.NewSingleFlight()
	wg.Add(round)
	for i := 0; i < round; i++ {
		go func() {
			defer wg.Done()
			val, err := singleFlight.Do("get_rand_number", func() (interface{}, error) {
				// 模拟耗时操作
				time.Sleep(200 * time.Millisecond)
				return rand.Int(), nil
			})
			if err != nil {
				fmt.Printf("error: %v\n", err)
			} else {
				fmt.Printf("result: %v\n", val)
			}
		}()
	}
	wg.Wait()
}
