package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

/*
陷阱 1：gorilla/websocket 并发写导致 panic

gorilla/websocket 的 Conn 不是并发安全的。
如果多个 goroutine 同时调用 WriteMessage，会触发 panic:
  "concurrent write to websocket connection"

这是最常见的 WebSocket 生产事故之一。
*/

// ❌ 错误示范：多个 goroutine 直接写同一个连接
func badHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := fmt.Sprintf("message from goroutine %d", id)
			// 💥 多个 goroutine 同时写 → panic
			conn.WriteMessage(websocket.TextMessage, []byte(msg))
		}(i)
	}
	wg.Wait()
}

// ✅ 正确做法：通过 channel 序列化写操作
func goodHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	writeCh := make(chan []byte, 256)

	// 单独的写 goroutine
	go func() {
		for msg := range writeCh {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := fmt.Sprintf("message from goroutine %d", id)
			writeCh <- []byte(msg) // 安全：通过 channel 发送
		}(i)
	}
	wg.Wait()
	close(writeCh)
}

func main() {
	http.HandleFunc("/bad", badHandler)
	http.HandleFunc("/good", goodHandler)
	log.Println("server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
