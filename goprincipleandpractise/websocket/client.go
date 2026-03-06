package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // 54s，必须小于 pongWait
	maxMessageSize = 4096
)

// Client 是 Hub 与 WebSocket 连接之间的中间层。
// 每个 Client 启动两个 goroutine：readPump 和 writePump。
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// readPump 从 WebSocket 连接中读取消息，转发到 Hub 的 broadcast channel。
// 每个连接只有一个 readPump goroutine。
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure) {
				log.Printf("read error: %v", err)
			}
			return
		}
		c.hub.broadcast <- message
	}
}

// writePump 从 send channel 中读取消息，写到 WebSocket 连接。
// 同时负责发送 Ping 帧维持心跳。
// 每个连接只有一个 writePump goroutine，保证写操作的串行化。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了 send channel
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs 处理 WebSocket 连接请求，将其升级后注册到 Hub。
func ServeWs(hub *Hub, upgrader websocket.Upgrader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}

		client := &Client{
			hub:  hub,
			conn: conn,
			send: make(chan []byte, 256),
		}
		hub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
