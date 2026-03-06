package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// NewUpgrader 创建一个 WebSocket 升级器。
// checkOrigin 控制是否校验 Origin，生产环境应传入严格的校验函数。
func NewUpgrader(checkOrigin func(r *http.Request) bool) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin,
	}
}

// EchoHandler 演示最基础的 WebSocket echo 服务。
// 收到什么消息就原样返回。
func EchoHandler(upgrader websocket.Upgrader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseNormalClosure) {
					log.Printf("read error: %v", err)
				}
				return
			}
			if err := conn.WriteMessage(msgType, msg); err != nil {
				log.Printf("write error: %v", err)
				return
			}
		}
	}
}
