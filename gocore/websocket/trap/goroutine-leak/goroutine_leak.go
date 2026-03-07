package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/gorilla/websocket"
)

/*
陷阱 2：WebSocket 连接 goroutine 泄漏

每个 WebSocket 连接通常启动 2 个 goroutine（readPump + writePump）。
如果客户端异常断开（如直接关闭浏览器、网络中断），而服务端没有：
1. 设置读超时（ReadDeadline）
2. 设置心跳检测（Ping/Pong）

那么 readPump 的 ReadMessage 会永远阻塞，goroutine 永远不会退出。
10 万个僵尸连接 = 20 万个泄漏的 goroutine + 对应的内存。
*/

// ❌ 错误示范：没有心跳和超时的连接处理
func leakyHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	// 没有 defer conn.Close()，如果下面的循环异常退出，连接不会关闭
	// 没有 SetReadDeadline，如果客户端断网，ReadMessage 永远阻塞

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// 只是 break，没有清理
			break
		}
		conn.WriteMessage(websocket.TextMessage, msg)
	}
	// goroutine 虽然退出了，但如果网络层没有检测到断开，
	// 上面的 ReadMessage 可能永远不会返回错误
}

// ✅ 正确做法：设置超时 + 心跳 + 优雅清理
func safeHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close() // 确保连接总是关闭

	const (
		pongWait   = 60 * time.Second
		pingPeriod = 54 * time.Second
		writeWait  = 10 * time.Second
	)

	// 设置读超时：如果 pongWait 时间内没有收到任何数据（包括 Pong），连接视为死亡
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// 启动心跳 goroutine
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
	close(done) // 通知心跳 goroutine 退出
}

func main() {
	http.HandleFunc("/leaky", leakyHandler)
	http.HandleFunc("/safe", safeHandler)

	// 定期打印 goroutine 数量，方便观察是否有泄漏
	go func() {
		for {
			time.Sleep(5 * time.Second)
			fmt.Printf("goroutine count: %d\n", runtime.NumGoroutine())
		}
	}()

	log.Println("server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
