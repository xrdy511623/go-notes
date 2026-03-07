package txbatchvsindividual

import (
	"fmt"
	"testing"
)

/*
对比事务批量 INSERT 与逐条 INSERT 的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .
*/

func benchmarkIndividual(b *testing.B, n int) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := InsertIndividual(db, n); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkBatchTx(b *testing.B, n int) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := InsertBatchTx(db, n); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkBatchTxPrepared(b *testing.B, n int) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := InsertBatchTxPrepared(db, n); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsert(b *testing.B) {
	sizes := []int{10, 100, 1000}
	for _, n := range sizes {
		b.Run(fmt.Sprintf("Individual/N=%d", n), func(b *testing.B) {
			benchmarkIndividual(b, n)
		})
		b.Run(fmt.Sprintf("BatchTx/N=%d", n), func(b *testing.B) {
			benchmarkBatchTx(b, n)
		})
		b.Run(fmt.Sprintf("BatchTxPrepared/N=%d", n), func(b *testing.B) {
			benchmarkBatchTxPrepared(b, n)
		})
	}
}
