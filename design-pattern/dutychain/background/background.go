package background

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// 假如我们有个 Handler 方法，是对 HTTP 请求进行处理的业务逻辑代码。

// Handler 定义实际的业务逻辑处理函数
func Handler(w http.ResponseWriter, r *http.Request) {
	// 业务逻辑处理
}

// 现在我们想给 Handler 加个接口响应时间统计功能，你可以直接在 Handler 方法内部加上这段逻辑。

// HandlerWithTimeCost 加个接口响应时间统计功能
func HandlerWithTimeCost(w http.ResponseWriter, r *http.Request) {
	// 记录请求开始时间
	start := time.Now()
	defer func() { // 计算响应时间
		elapsed := time.Since(start) // 打印响应时间
		log.Printf("响应时间: %v\n", elapsed)
	}()
	fmt.Fprintf(w, "Hello, World!")
}

/*
那假如我们想给 HTTP 服务的所有接口加上响应时间统计功能呢？如果继续把这段逻辑放在接口的实现逻辑里，
我们需要把这段代码复制到各个业务处理函数，会造成大量的代码重复。而且，后续如果需要给所有接口加上鉴权
等更多功能，所有的接口实现函数又得改一遍。怎么才能避免这种情况呢？我们可以用责任链模式实现给所有接口
加上统一的处理逻辑。责任链模式允许你将请求沿着处理者链进行发送。收到请求后，每个处理者均可对请求进行处理，
或将其传递给链上的下个处理者。
*/
