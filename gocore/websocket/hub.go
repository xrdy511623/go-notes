package websocket

// Hub 维护活跃客户端集合，并负责消息广播。
// 所有对 clients 的操作都在 Run 的单一 goroutine 中完成，无需加锁。
type Hub struct {
	clients    map[*Client]struct{}
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// 发送缓冲区满，客户端处理不过来，断开连接
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// ClientCount 返回当前连接数（仅用于测试和监控）。
// 注意：这个方法在 Hub.Run goroutine 外部调用时存在竞态，
// 只适合粗略监控，不保证精确。
func (h *Hub) ClientCount() int {
	return len(h.clients)
}
