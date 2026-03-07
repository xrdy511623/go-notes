package preparedvsunprepared

import (
	"fmt"
	"testing"
)

/*
对比 Prepared Statement 与直接执行的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .
*/

func BenchmarkInsertPrepared(b *testing.B) {
	d, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := d.InsertPrepared("test_value"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertUnprepared(b *testing.B) {
	d, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	b.ResetTimer()
	for b.Loop() {
		if err := d.InsertUnprepared("test_value"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertPreparedParallel(b *testing.B) {
	d, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := d.InsertPrepared("test_value"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkInsertUnpreparedParallel(b *testing.B) {
	d, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := d.InsertUnprepared("test_value"); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkInsertLoopPrepared(b *testing.B) {
	d, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < 100; j++ {
			if err := d.InsertPrepared(fmt.Sprintf("val_%d", j)); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkInsertLoopUnprepared(b *testing.B) {
	d, err := NewDB()
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()

	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < 100; j++ {
			if err := d.InsertUnprepared(fmt.Sprintf("val_%d", j)); err != nil {
				b.Fatal(err)
			}
		}
	}
}
