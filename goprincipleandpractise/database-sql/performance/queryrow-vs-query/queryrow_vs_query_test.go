package queryrowvsquery

import (
	"sync/atomic"
	"testing"
)

/*
对比 QueryRow 与 Query 获取单行数据的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .
*/

func BenchmarkQueryRow(b *testing.B) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var seq atomic.Int64
	b.ResetTimer()
	for b.Loop() {
		id := int(seq.Add(1)%1000) + 1
		if _, err := FetchWithQueryRow(db, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQuery(b *testing.B) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var seq atomic.Int64
	b.ResetTimer()
	for b.Loop() {
		id := int(seq.Add(1)%1000) + 1
		if _, err := FetchWithQuery(db, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryRowParallel(b *testing.B) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var seq atomic.Int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := int(seq.Add(1)%1000) + 1
			if _, err := FetchWithQueryRow(db, id); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkQueryParallel(b *testing.B) {
	db, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var seq atomic.Int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := int(seq.Add(1)%1000) + 1
			if _, err := FetchWithQuery(db, id); err != nil {
				b.Fatal(err)
			}
		}
	})
}
