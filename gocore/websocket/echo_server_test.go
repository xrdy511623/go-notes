package websocket

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

func allowAllOrigins(r *http.Request) bool { return true }

func TestEchoHandler(t *testing.T) {
	upgrader := NewUpgrader(allowAllOrigins)
	server := httptest.NewServer(EchoHandler(upgrader))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	tests := []struct {
		name    string
		msgType int
		payload string
	}{
		{"text message", websocket.TextMessage, "hello websocket"},
		{"empty message", websocket.TextMessage, ""},
		{"unicode message", websocket.TextMessage, "你好 WebSocket 🌐"},
		{"binary message", websocket.BinaryMessage, "\x00\x01\x02\x03"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer conn.Close()

			if err := conn.WriteMessage(tt.msgType, []byte(tt.payload)); err != nil {
				t.Fatalf("write: %v", err)
			}

			gotType, got, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if gotType != tt.msgType {
				t.Errorf("message type: got %d, want %d", gotType, tt.msgType)
			}
			if string(got) != tt.payload {
				t.Errorf("payload: got %q, want %q", got, tt.payload)
			}
		})
	}
}

func TestEchoMultipleMessages(t *testing.T) {
	upgrader := NewUpgrader(allowAllOrigins)
	server := httptest.NewServer(EchoHandler(upgrader))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	for i := 0; i < 100; i++ {
		msg := fmt.Sprintf("message-%d", i)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
			t.Fatalf("write %d: %v", i, err)
		}
		_, got, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if string(got) != msg {
			t.Errorf("message %d: got %q, want %q", i, got, msg)
		}
	}
}

func TestConcurrentConnections(t *testing.T) {
	upgrader := NewUpgrader(allowAllOrigins)
	server := httptest.NewServer(EchoHandler(upgrader))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	const numClients = 50

	var wg sync.WaitGroup
	wg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func(id int) {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Errorf("client %d: dial: %v", id, err)
				return
			}
			defer conn.Close()

			msg := fmt.Sprintf("hello from client %d", id)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				t.Errorf("client %d: write: %v", id, err)
				return
			}

			_, got, err := conn.ReadMessage()
			if err != nil {
				t.Errorf("client %d: read: %v", id, err)
				return
			}
			if string(got) != msg {
				t.Errorf("client %d: got %q, want %q", id, got, msg)
			}
		}(i)
	}
	wg.Wait()
}

func TestChatBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	upgrader := NewUpgrader(allowAllOrigins)
	server := httptest.NewServer(ServeWs(hub, upgrader))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	const numClients = 5

	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client %d dial: %v", i, err)
		}
		defer conn.Close()
		conns[i] = conn
	}

	// 等待所有客户端注册完成
	// 客户端 0 发送消息
	msg := "broadcast test"
	if err := conns[0].WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}

	// 所有客户端（包括发送者）应收到广播
	for i, conn := range conns {
		_, got, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d read: %v", i, err)
		}
		if string(got) != msg {
			t.Errorf("client %d: got %q, want %q", i, got, msg)
		}
	}
}
