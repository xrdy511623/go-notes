package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"go-notes/designpattern/builderandoption/builder"
	"go-notes/designpattern/builderandoption/option"
	"go-notes/designpattern/dutychain/auth"
	"go-notes/designpattern/dutychain/logging"
	"go-notes/designpattern/dutychain/middleware"
	"go-notes/designpattern/dutychain/recovery"
	"go-notes/designpattern/observer/eventbus"
	"go-notes/designpattern/observer/logger"
	"go-notes/designpattern/observer/metrics"
	"go-notes/designpattern/observer/notifier"
	_ "go-notes/designpattern/registerfactory/dingding"
	_ "go-notes/designpattern/registerfactory/feishu"
	"go-notes/designpattern/registerfactory/sender"
	_ "go-notes/designpattern/registerfactory/weixin"
)

func main() {
	// 注册工厂模式示例
	s, err := sender.New("dingding")
	if err != nil {
		fmt.Printf("sender.New err: %v\n", err)
		return
	}
	if err := s.Send("Hello"); err != nil {
		fmt.Printf("sender.Send err: %v\n", err)
	}

	// 观察者模式示例
	bus := eventbus.New()

	log := logger.New("APP")
	met := metrics.New("order")
	ntf := notifier.New("email")

	bus.Subscribe("order.created", log)
	bus.Subscribe("order.created", met)
	bus.Subscribe("order.created", ntf)

	fmt.Println("\n--- 发布第一个事件 ---")
	bus.Publish(eventbus.Event{Topic: "order.created", Payload: "order-123"})
	fmt.Printf("指标计数: %d\n", met.Count())

	fmt.Println("\n--- 取消通知观察者，发布第二个事件 ---")
	bus.Unsubscribe("order.created", ntf.Name())
	bus.Publish(eventbus.Event{Topic: "order.created", Payload: "order-456"})
	fmt.Printf("指标计数: %d\n", met.Count())

	// 责任链模式示例（HTTP 中间件）
	fmt.Println("\n--- 责任链模式（HTTP 中间件） ---")
	chain := middleware.Chain(
		logging.New(),
		auth.New("valid_token"),
		recovery.New(),
	)
	handler := chain(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	req.Header.Set("Authorization", "valid_token")
	rec := httptest.NewRecorder()
	handler(rec, req)
	fmt.Printf("Response: %s (status %d)\n", rec.Body.String(), rec.Code)

	req2 := httptest.NewRequest("GET", "/hello", nil)
	rec2 := httptest.NewRecorder()
	handler(rec2, req2)
	fmt.Printf("Response: %s (status %d)\n", rec2.Body.String(), rec2.Code)

	// 构建者模式示例
	fmt.Println("\n--- Builder 模式 ---")
	srv1, err := builder.NewBuilder("0.0.0.0", 443).
		ReadTimeout(10*time.Second).
		MaxConnections(5000).
		TLS("cert.pem", "key.pem").
		Build()
	if err != nil {
		fmt.Printf("builder err: %v\n", err)
	} else {
		fmt.Printf("Builder: %s:%d (TLS=%v, MaxConn=%d)\n",
			srv1.Host, srv1.Port, srv1.TLSEnabled, srv1.MaxConnections)
	}

	// Functional Options 模式示例
	fmt.Println("\n--- Functional Options 模式 ---")
	srv2, err := option.NewServer("0.0.0.0", 443,
		option.WithReadTimeout(10*time.Second),
		option.WithMaxConnections(5000),
		option.WithTLS("cert.pem", "key.pem"),
	)
	if err != nil {
		fmt.Printf("option err: %v\n", err)
	} else {
		fmt.Printf("Option: %s:%d (TLS=%v, MaxConn=%d)\n",
			srv2.Host, srv2.Port, srv2.TLSEnabled, srv2.MaxConnections)
	}
}
