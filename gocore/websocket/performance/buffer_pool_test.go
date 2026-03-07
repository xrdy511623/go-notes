package performance

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func benchmarkEcho(b *testing.B, upgrader websocket.Upgrader, msgSize int) {
	b.Helper()
	server := httptest.NewServer(echoHandlerWithUpgrader(upgrader))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	msg := make([]byte, msgSize)
	for i := range msg {
		msg[i] = 'A'
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			b.Fatalf("write: %v", err)
		}
		if _, _, err := conn.ReadMessage(); err != nil {
			b.Fatalf("read: %v", err)
		}
	}
}

func BenchmarkNoPool_128B(b *testing.B) {
	benchmarkEcho(b, NewUpgraderNoPool(), 128)
}

func BenchmarkWithPool_128B(b *testing.B) {
	benchmarkEcho(b, NewUpgraderWithPool(), 128)
}

func BenchmarkNoPool_4KB(b *testing.B) {
	benchmarkEcho(b, NewUpgraderNoPool(), 4096)
}

func BenchmarkWithPool_4KB(b *testing.B) {
	benchmarkEcho(b, NewUpgraderWithPool(), 4096)
}

// BenchmarkMultiConn 模拟多连接并发收发场景，
// 测试缓冲区复用在多连接同时活跃时的效果。
func benchmarkMultiConn(b *testing.B, upgrader websocket.Upgrader, numConns int) {
	b.Helper()
	server := httptest.NewServer(echoHandlerWithUpgrader(upgrader))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conns := make([]*websocket.Conn, numConns)
	for i := range conns {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatalf("dial %d: %v", i, err)
		}
		conns[i] = conn
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	msg := []byte("hello")
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := conns[i%numConns]
		if err := c.WriteMessage(websocket.TextMessage, msg); err != nil {
			b.Fatalf("write: %v", err)
		}
		if _, _, err := c.ReadMessage(); err != nil {
			b.Fatalf("read: %v", err)
		}
	}
}

func BenchmarkMultiConn_NoPool_50(b *testing.B) {
	benchmarkMultiConn(b, NewUpgraderNoPool(), 50)
}

func BenchmarkMultiConn_WithPool_50(b *testing.B) {
	benchmarkMultiConn(b, NewUpgraderWithPool(), 50)
}
