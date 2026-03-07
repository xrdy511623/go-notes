package performance

import (
	"sync"
)

var bufSizeSink int64

// ProducerConsumer 单生产者→单消费者，使用指定缓冲区大小的 channel
func ProducerConsumer(n, bufSize int) int64 {
	ch := make(chan int64, bufSize)

	go func() {
		for i := range n {
			ch <- int64(i)
		}
		close(ch)
	}()

	var sum int64
	for v := range ch {
		sum += v
	}
	return sum
}

// MultiProducerConsumer M个生产者→1个消费者
func MultiProducerConsumer(producers, opsPerProducer, bufSize int) int64 {
	ch := make(chan int64, bufSize)
	var wg sync.WaitGroup
	wg.Add(producers)

	for p := range producers {
		go func() {
			defer wg.Done()
			base := int64(p * opsPerProducer)
			for i := range opsPerProducer {
				ch <- base + int64(i)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var sum int64
	for v := range ch {
		sum += v
	}
	return sum
}
