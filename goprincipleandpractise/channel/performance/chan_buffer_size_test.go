package performance

import (
	"fmt"
	"testing"
)

/*
Channel 缓冲区大小对吞吐量的影响

执行命令:

	go test -run '^$' -bench 'BufSize' -benchtime=3s -count=3 -benchmem .

对比维度:
  1. 单生产者→单消费者: 缓冲区 0/1/10/100/1000
  2. 4生产者→1消费者: 缓冲区 0/1/10/100/1000

结论:
  - 无缓冲 channel（size=0）每次 send 都会阻塞等待 receive，吞吐量最低
  - 缓冲区为 1 时，吞吐量显著提升（减少了同步等待）
  - 缓冲区从 1 增长到 100 时，吞吐量持续提升但增幅递减
  - 超过一定大小后（通常 100-1000），收益趋近于零，反而浪费内存

选型建议:
  - 同步语义（确认对方已处理）→ 无缓冲 channel
  - 解耦生产者和消费者的速度差异 → 缓冲区 = 预期突发量
  - 高吞吐量管道 → 缓冲区 64-256 通常是较好的平衡点
  - 避免使用超大缓冲区来掩盖消费者处理过慢的问题
*/

var bufSizes = []int{0, 1, 10, 100, 1000}

// ------------------- 单生产者→单消费者 -------------------

func BenchmarkBufSizeSingleProducer(b *testing.B) {
	for _, size := range bufSizes {
		b.Run(fmt.Sprintf("buf=%d", size), func(b *testing.B) {
			for b.Loop() {
				bufSizeSink = ProducerConsumer(10000, size)
			}
		})
	}
}

// ------------------- 多生产者→单消费者 -------------------

func BenchmarkBufSizeMultiProducer(b *testing.B) {
	for _, size := range bufSizes {
		b.Run(fmt.Sprintf("buf=%d", size), func(b *testing.B) {
			for b.Loop() {
				bufSizeSink = MultiProducerConsumer(4, 2500, size)
			}
		})
	}
}
