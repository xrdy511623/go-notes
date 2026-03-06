package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

/*
陷阱 3：跨站 WebSocket 劫持（Cross-Site WebSocket Hijacking, CSWSH）

WebSocket 连接不受同源策略保护。如果服务器不校验 Origin 头，
恶意网站可以在用户不知情的情况下，利用用户的浏览器 Cookie
连接到你的 WebSocket 服务器，窃取实时数据或执行操作。

攻击场景：
1. 用户登录了 app.example.com，浏览器存储了 session cookie
2. 用户访问了恶意网站 evil.com
3. evil.com 的 JavaScript 发起 ws://app.example.com/ws 连接
4. 浏览器自动携带 app.example.com 的 cookie
5. 如果服务器不检查 Origin，连接建立成功
6. 攻击者可以通过这个连接读取用户的私密数据

gorilla/websocket 默认行为：CheckOrigin 默认会拒绝跨域请求
（即 Origin 头与 Host 头不匹配时返回 false）。

但很多教程为了"简化"，直接写 CheckOrigin: func(r *http.Request) bool { return true }，
这在生产环境是严重的安全隐患。
*/

// ❌ 危险：允许任意来源的连接
func unsafeUpgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 任何网站都可以连接！
		},
	}
}

// ✅ 安全：白名单校验 Origin
func safeUpgrader(allowedOrigins []string) websocket.Upgrader {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[strings.ToLower(origin)] = struct{}{}
	}

	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				// 非浏览器客户端可能没有 Origin 头
				// 根据业务需求决定是否允许
				return false
			}
			_, ok := allowed[strings.ToLower(origin)]
			return ok
		},
	}
}

func wsHandler(upgrader websocket.Upgrader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			fmt.Printf("received: %s\n", msg)
		}
	}
}

func main() {
	// 生产环境应使用白名单
	upgrader := safeUpgrader([]string{
		"https://yourdomain.com",
		"https://www.yourdomain.com",
	})

	http.HandleFunc("/ws", wsHandler(upgrader))
	log.Println("server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
