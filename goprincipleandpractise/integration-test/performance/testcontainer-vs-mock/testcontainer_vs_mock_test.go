package testcontainer_vs_mock

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// 对比 Mock 方式和模拟 Testcontainer 方式的测试执行速度
//
// 运行方式:
//   go test -run='^$' -bench=. -benchmem -benchtime=3s .
//
// 预期结果:
//   Mock 方式比 Testcontainer 方式快 10-100 倍
//   但 Mock 无法验证真实 SQL 语义，两者应配合使用

func BenchmarkMock_CreateAndFind(b *testing.B) {
	repo := NewMockUserRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user_%d@example.com", i)

		id, err := repo.Create(ctx, name, email)
		if err != nil {
			b.Fatal(err)
		}

		_, _, err = repo.FindByID(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimulatedDB_CreateAndFind(b *testing.B) {
	// 模拟真实数据库的网络延迟（本地容器约 0.5-2ms）
	repo := NewSimulatedDBRepository(500 * time.Microsecond)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user_%d@example.com", i)

		id, err := repo.Create(ctx, name, email)
		if err != nil {
			b.Fatal(err)
		}

		_, _, err = repo.FindByID(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSimulatedDB_RemoteLatency 模拟远程数据库的延迟
func BenchmarkSimulatedDB_RemoteLatency(b *testing.B) {
	// 模拟远程数据库延迟（约 5-20ms）
	repo := NewSimulatedDBRepository(5 * time.Millisecond)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("user_%d", i)
		email := fmt.Sprintf("user_%d@example.com", i)

		id, err := repo.Create(ctx, name, email)
		if err != nil {
			b.Fatal(err)
		}

		_, _, err = repo.FindByID(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
	}
}
