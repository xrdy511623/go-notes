package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

/*
我们可以用责任链模式实现给所有接口加上统一的处理逻辑。责任链模式允许你将请求沿着处理者链进行发送。收到请求后，
每个处理者均可对请求进行处理，或将其传递给链上的下个处理者。下面的代码定义了耗时统计函数 CostMiddleware 和
鉴权函数 AuthMiddleware，它们接收请求处理函数类型的参数 next，并且返回了一个新的请求处理函数。这个新的请求
处理函数的内部实现是，在调用 next 函数之前，先执行耗时统计和鉴权的逻辑。封装返回的请求处理函数就是责任链中的
处理者节点。

实现责任链中的处理者节点后，用了 ApplyMiddleware 方法，将传入的业务逻辑处理函数和耗时统计函数、鉴权函数
构造成责任链，并将责任链作为 http 请求的处理函数，就能给所有接口加上耗时统计和权限验证功能。
*/

// Middleware 定义一个函数类型，用于表示中间件函数
type Middleware func(http.HandlerFunc) http.HandlerFunc

// CostMiddleware 接口耗时记录中间件
func CostMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() { // 计算响应时间
			elapsed := time.Since(start) // 打印响应时间
			log.Printf("响应时间: %v\n", elapsed)
		}()
		next(w, r)
	}
}

// AuthMiddleware 权限验证中间件
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 这里简单模拟权限验证，假设只有特定的用户可以访问
		if r.Header.Get("Authorization") != "valid_token" {
			http.Error(w, "权限不足", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// Handler 定义实际的业务逻辑处理函数
func Handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World!")
}

// ApplyMiddleware 应用中间件到处理函数的函数
func ApplyMiddleware(middlewares []Middleware, handler http.HandlerFunc) http.HandlerFunc {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}

func main() {
	// 创建一个HTTP服务器
	http.HandleFunc("/", ApplyMiddleware(
		[]Middleware{CostMiddleware, AuthMiddleware},
		Handler,
	))

	// 启动服务器并监听端口
	log.Fatal(http.ListenAndServe(":8080", nil))
}
