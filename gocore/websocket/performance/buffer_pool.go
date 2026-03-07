package performance

import (
	"bytes"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

/*
WebSocket 连接的缓冲区复用对比实验。

每个 WebSocket 连接都需要独立的读/写缓冲区，在大量连接场景下，
缓冲区的内存分配和 GC 压力是主要的性能瓶颈之一。

gorilla/websocket 支持通过 WriteBufferPool 复用写缓冲区。
本实验对比以下两种方式的内存分配差异：

1. NoPool：每个连接独立分配缓冲区（默认行为）
2. WithPool：使用 sync.Pool 复用写缓冲区

运行：
go test -bench='^Benchmark(NoPool|WithPool)' -benchmem -benchtime=3s .
*/

type bufferPool struct {
	pool sync.Pool
}

func (p *bufferPool) Get() interface{} {
	return p.pool.Get()
}

func (p *bufferPool) Put(v interface{}) {
	if buf, ok := v.(*bytes.Buffer); ok {
		buf.Reset()
		p.pool.Put(v)
	}
}

var sharedBufferPool = &bufferPool{
	pool: sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 0, 1024))
		},
	},
}

func echoHandlerWithUpgrader(upgrader websocket.Upgrader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		for {
			msgType, msg, readErr := conn.ReadMessage()
			if readErr != nil {
				return
			}
			if writeErr := conn.WriteMessage(msgType, msg); writeErr != nil {
				return
			}
		}
	}
}

func NewUpgraderNoPool() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
}

func NewUpgraderWithPool() websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		WriteBufferPool: sharedBufferPool,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
}
