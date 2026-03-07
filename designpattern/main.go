package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"time"

	"go-notes/designpattern/adapter/logadapter"
	"go-notes/designpattern/adapter/matcher"
	"go-notes/designpattern/builderandoption/builder"
	"go-notes/designpattern/builderandoption/option"
	"go-notes/designpattern/circuitbreaker/breaker"
	"go-notes/designpattern/circuitbreaker/ratelimiter"
	"go-notes/designpattern/circuitbreaker/resilience"
	"go-notes/designpattern/command/editor"
	"go-notes/designpattern/command/transaction"
	"go-notes/designpattern/composite/fstree"
	"go-notes/designpattern/composite/validator"
	"go-notes/designpattern/decorator/reader"
	"go-notes/designpattern/decorator/responsewriter"
	"go-notes/designpattern/dutychain/auth"
	"go-notes/designpattern/dutychain/logging"
	"go-notes/designpattern/dutychain/middleware"
	"go-notes/designpattern/dutychain/recovery"
	"go-notes/designpattern/fanoutin/mapreduce"
	"go-notes/designpattern/fanoutin/parallel"
	"go-notes/designpattern/fanoutin/race"
	"go-notes/designpattern/flyweight/color"
	"go-notes/designpattern/flyweight/icon"
	"go-notes/designpattern/flyweight/intern"
	"go-notes/designpattern/flyweight/style"
	"go-notes/designpattern/iterator/pipeline"
	"go-notes/designpattern/iterator/sequence"
	"go-notes/designpattern/objectpool/bufferpool"
	"go-notes/designpattern/objectpool/workerpool"
	"go-notes/designpattern/observer/eventbus"
	"go-notes/designpattern/observer/logger"
	"go-notes/designpattern/observer/metrics"
	"go-notes/designpattern/observer/notifier"
	"go-notes/designpattern/pipeline/runner"
	"go-notes/designpattern/pipeline/stage"
	"go-notes/designpattern/proxy/cacheproxy"
	"go-notes/designpattern/proxy/fetcher"
	"go-notes/designpattern/proxy/metricsproxy"
	"go-notes/designpattern/proxy/protectproxy"
	_ "go-notes/designpattern/registerfactory/dingding"
	_ "go-notes/designpattern/registerfactory/feishu"
	"go-notes/designpattern/registerfactory/sender"
	_ "go-notes/designpattern/registerfactory/weixin"
	"go-notes/designpattern/singleton/configmgr"
	"go-notes/designpattern/singleton/connpool"
	connfsm "go-notes/designpattern/statemachine/conn"
	"go-notes/designpattern/statemachine/fsm"
	"go-notes/designpattern/statemachine/order"
	"go-notes/designpattern/strategy/payment"
	"go-notes/designpattern/strategy/retry"
	"go-notes/designpattern/templatemethod/csvexport"
	"go-notes/designpattern/templatemethod/export"
	"go-notes/designpattern/templatemethod/mdexport"
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

	// 单例模式示例
	fmt.Println("\n--- 单例模式（sync.Once 懒加载） ---")
	cfg, err := configmgr.Instance()
	if err != nil {
		fmt.Printf("configmgr err: %v\n", err)
	} else {
		fmt.Printf("Config: app=%s port=%d debug=%v\n", cfg.AppName, cfg.ServerPort, cfg.Debug)
	}
	cfg2, _ := configmgr.Instance()
	fmt.Printf("同一实例: %v\n", cfg == cfg2)

	fmt.Println("\n--- 单例模式（饿汉式连接池） ---")
	pool := connpool.Instance()
	conn := pool.Get()
	fmt.Printf("ConnPool: got %s, idle=%d\n", conn, pool.Len())
	pool.Put(conn)
	fmt.Printf("ConnPool: returned, idle=%d\n", pool.Len())

	// 装饰器模式示例（io.Reader 层层包装）
	fmt.Println("\n--- 装饰器模式（io.Reader 层层包装） ---")
	data := "The Decorator Pattern in Go: wrapping io.Reader layer by layer"
	base := strings.NewReader(data)
	progress := reader.NewProgressReader(base, int64(len(data)), func(bytesRead, total int64) {
		fmt.Printf("  进度: %d/%d (%.0f%%)\n", bytesRead, total, float64(bytesRead)/float64(total)*100)
	})
	counting := reader.NewCountingReader(progress)
	result, _ := io.ReadAll(counting)
	fmt.Printf("读取完成: %q\n总字节数: %d\n", string(result), counting.BytesRead())

	// 装饰器模式示例（http.ResponseWriter 包装）
	fmt.Println("\n--- 装饰器模式（http.ResponseWriter 包装） ---")
	metricsMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			rr := responsewriter.NewResponseRecorder(w)
			next(rr, r)
			fmt.Printf("  [%s] %s → %d (%d bytes)\n",
				r.Method, r.URL.Path, rr.StatusCode(), rr.BytesWritten())
		}
	}
	decoratedHandler := metricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":"order-789","status":"created"}`)
	})
	req3 := httptest.NewRequest("POST", "/api/orders", nil)
	rec3 := httptest.NewRecorder()
	decoratedHandler(rec3, req3)
	fmt.Printf("Response: %s (status %d)\n", rec3.Body.String(), rec3.Code)

	// 策略模式示例（支付策略：接口方式，运行时切换）
	fmt.Println("\n--- 策略模式（支付策略 — 接口方式） ---")
	co := payment.NewCheckout(&payment.Alipay{MerchantID: "M001"})
	txn, err := co.Pay(99.99)
	if err != nil {
		fmt.Printf("pay err: %v\n", err)
	} else {
		fmt.Printf("支付宝: %s\n", txn)
	}

	co.SetStrategy(&payment.WechatPay{AppID: "wx456"})
	txn, err = co.Pay(99.99)
	if err != nil {
		fmt.Printf("pay err: %v\n", err)
	} else {
		fmt.Printf("微信支付: %s\n", txn)
	}

	co.SetStrategy(payment.StrategyFunc(func(amount float64) (string, error) {
		return fmt.Sprintf("MOCK-%.2f", amount), nil
	}))
	txn, _ = co.Pay(99.99)
	fmt.Printf("函数策略: %s\n", txn)

	// 迭代器模式示例（range-over-func / iter.Seq）
	fmt.Println("\n--- 迭代器模式（range-over-func / iter.Seq） ---")
	set := sequence.New(5, 3, 8, 1, 9, 2, 7, 4, 6)
	fmt.Printf("有序集合: ")
	for v := range set.All() {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	fmt.Printf("逆序遍历: ")
	for v := range set.Backward() {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	fmt.Printf("区间 [3,7): ")
	for v := range set.Range(3, 7) {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	fmt.Printf("索引对:  ")
	for i, v := range set.Pairs() {
		fmt.Printf("[%d]=%d ", i, v)
	}
	fmt.Println()

	// 迭代器管道组合
	fmt.Println("\n--- 迭代器管道（Filter → Map → Take） ---")
	nums := sequence.New(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	pipelined := pipeline.Take(
		pipeline.Map(
			pipeline.Filter(nums.All(), func(v int) bool { return v%2 == 0 }),
			func(v int) int { return v * 10 },
		),
		3,
	)
	fmt.Printf("偶数×10取前3: ")
	for v := range pipelined {
		fmt.Printf("%d ", v)
	}
	fmt.Println()

	// 对象池模式示例（sync.Pool 缓冲区复用）
	fmt.Println("\n--- 对象池模式（sync.Pool 缓冲区复用） ---")
	bp := bufferpool.New(256)
	buf := bp.Get()
	buf.WriteString("pooled buffer content")
	fmt.Printf("Buffer: %q (cap=%d)\n", buf.String(), buf.Cap())
	bp.Put(buf)

	buf2 := bp.Get()
	fmt.Printf("复用后: len=%d cap>=%d (已 Reset)\n", buf2.Len(), buf2.Cap())
	bp.Put(buf2)

	// 对象池模式示例（有界工作池）
	fmt.Println("\n--- 对象池模式（有界工作池） ---")
	wp := workerpool.New(3)
	for i := range 5 {
		wp.Submit(func() {
			fmt.Printf("  worker 完成任务 %d\n", i)
		})
	}
	wp.Close()
	fmt.Println("工作池已关闭，所有 worker 已退出")

	// 适配器模式示例（函数类型适配器 — HandlerFunc 模式）
	fmt.Println("\n--- 适配器模式（函数类型适配器 — HandlerFunc 模式） ---")
	m := matcher.All(
		matcher.HasPrefix("Go"),
		matcher.Contains("1."),
	)
	for _, s := range []string{"Go 1.24", "Go 语言", "Rust 1.0"} {
		fmt.Printf("  %q → match=%v\n", s, m.Match(s))
	}

	dateMatcher, _ := matcher.NewRegexpMatcher(`^\d{4}-\d{2}-\d{2}$`)
	fmt.Printf("  日期匹配 \"2024-01-15\" → %v\n", dateMatcher.Match("2024-01-15"))

	// 适配器模式示例（结构体适配器 — Logger 桥接）
	fmt.Println("\n--- 适配器模式（结构体适配器 — Logger 桥接） ---")
	adapted := logadapter.FromStd(stdlog.New(os.Stdout, "", 0))
	adapted.Info("stdlib *log.Logger 适配为 Logger 接口, version=%s", "1.0")
	adapted.Error("错误日志也通过同一适配器输出")

	// 代理模式示例（缓存代理 — 控制访问，缓存命中时不调用真实主体）
	fmt.Println("\n--- 代理模式（缓存代理） ---")
	slow := &fetcher.SlowFetcher{Latency: 10 * time.Millisecond}
	cached := cacheproxy.New(slow, 5*time.Second)

	start := time.Now()
	v1, _ := cached.Fetch(context.Background(), "user:1")
	fmt.Printf("  首次获取: %q (耗时 %v)\n", v1, time.Since(start))

	start = time.Now()
	v2, _ := cached.Fetch(context.Background(), "user:1")
	fmt.Printf("  缓存命中: %q (耗时 %v)\n", v2, time.Since(start))
	fmt.Printf("  真实主体调用次数: %d\n", slow.Calls())

	// 代理模式示例（保护代理 — 未授权时拒绝，不调用真实主体）
	fmt.Println("\n--- 代理模式（保护代理） ---")
	protected := protectproxy.New(slow, "admin", "editor")

	adminCtx := protectproxy.WithRole(context.Background(), "admin")
	v3, _ := protected.Fetch(adminCtx, "secret")
	fmt.Printf("  admin 访问: %q\n", v3)

	guestCtx := protectproxy.WithRole(context.Background(), "guest")
	_, proxyErr := protected.Fetch(guestCtx, "secret")
	fmt.Printf("  guest 访问: %v\n", proxyErr)

	// 代理模式示例（度量代理 — 记录调用次数和平均延迟）
	fmt.Println("\n--- 代理模式（度量代理） ---")
	monitored := metricsproxy.New(cached)
	for _, key := range []string{"a", "b", "a", "c"} {
		monitored.Fetch(context.Background(), key)
	}
	fmt.Printf("  总调用: %d, 错误: %d, 平均延迟: %v\n",
		monitored.Calls(), monitored.Errors(), monitored.AvgLatency())

	// 熔断器模式示例（状态机: Closed → Open → HalfOpen → Closed）
	fmt.Println("\n--- 熔断器模式（状态机切换） ---")
	cb := breaker.New(breaker.Settings{
		MaxFailures: 3,
		Timeout:     20 * time.Millisecond,
		OnStateChange: func(from, to breaker.State) {
			fmt.Printf("  [状态切换] %s → %s\n", from, to)
		},
	})
	for i := 1; i <= 3; i++ {
		cb.Do(func() error { return errors.New("timeout") })
	}
	fmt.Printf("  连续 3 次失败后: state=%s\n", cb.State())

	err = cb.Do(func() error { return nil })
	fmt.Printf("  熔断中请求被拒: %v\n", err)

	time.Sleep(25 * time.Millisecond)
	fmt.Printf("  等待超时后: state=%s\n", cb.State())
	cb.Do(func() error { return nil })
	fmt.Printf("  探测成功，恢复: state=%s\n", cb.State())

	// 限流器示例（令牌桶）
	fmt.Println("\n--- 限流器（令牌桶 — 微服务治理配套） ---")
	lim := ratelimiter.New(10, 3)
	for i := 1; i <= 5; i++ {
		fmt.Printf("  请求 %d: allowed=%v\n", i, lim.Allow())
	}

	// 三板斧组合（限流 + 熔断 + 重试）
	fmt.Println("\n--- 微服务治理三板斧（限流 + 熔断 + 重试） ---")
	cb.Reset()
	callCount := 0
	err = resilience.Do(context.Background(), resilience.Config{
		Breaker:    cb,
		Limiter:    ratelimiter.New(100, 10),
		MaxRetries: 3,
		Backoff:    func(int) time.Duration { return time.Millisecond },
	}, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("transient error")
		}
		return nil
	})
	fmt.Printf("  结果: err=%v, 实际调用 %d 次（重试 2 次后成功）\n", err, callCount)

	// 模板方法模式示例（同一模板 Export()，不同 Exporter 实现）
	fmt.Println("\n--- 模板方法模式（同一算法骨架，不同导出格式） ---")
	exportRecords := []export.Record{
		{Fields: map[string]string{"name": "Alice", "age": "30", "city": "NYC"}},
		{Fields: map[string]string{"name": "Bob", "age": "25", "city": "LA"}},
	}
	exportColumns := []string{"name", "age", "city"}

	csvOut, _ := export.Export(csvexport.New(), exportColumns, exportRecords)
	fmt.Printf("CSV:\n%s", csvOut)

	mdOut, _ := export.Export(mdexport.New(), exportColumns, exportRecords)
	fmt.Printf("Markdown:\n%s", mdOut)

	// 状态机模式示例（订单工作流 FSM）
	fmt.Println("\n--- 状态机模式（订单工作流） ---")
	orderFSM, err := order.New()
	if err != nil {
		fmt.Printf("order.New err: %v\n", err)
		return
	}
	orderFSM.OnTransition(func(event fsm.Event, from, to fsm.State) {
		fmt.Printf("  [%s] %s → %s\n", event, from, to)
	})
	orderFSM.Fire(order.Pay)
	orderFSM.Fire(order.Ship)
	orderFSM.Fire(order.Deliver)
	fmt.Printf("  最终状态: %s\n", orderFSM.Current())

	orderFSM2, _ := order.New()
	orderFSM2.Fire(order.Pay)
	cancelErr := orderFSM2.Fire(order.Ship)
	fmt.Printf("  已付款→发货: err=%v, state=%s\n", cancelErr, orderFSM2.Current())
	cancelErr = orderFSM2.Fire(order.Cancel)
	fmt.Printf("  已发货→取消: err=%v（已发货不可取消）\n", cancelErr)

	// 状态机模式示例（网络连接 FSM — 对标 net/http ConnState）
	fmt.Println("\n--- 状态机模式（网络连接生命周期） ---")
	connFSM, err := connfsm.New()
	if err != nil {
		fmt.Printf("conn.New err: %v\n", err)
		return
	}
	connFSM.OnEnter(connfsm.Active, func(_ fsm.Event, _, _ fsm.State) {
		fmt.Println("  → 连接激活，开始处理请求")
	})
	connFSM.OnEnter(connfsm.Closed, func(_ fsm.Event, _, _ fsm.State) {
		fmt.Println("  → 连接关闭，释放资源")
	})
	connFSM.Fire(connfsm.Open)
	fmt.Printf("  可用事件: %v\n", connFSM.AvailableEvents())
	connFSM.Fire(connfsm.Park)
	fmt.Printf("  回到 idle: %s\n", connFSM.Current())
	connFSM.Fire(connfsm.Close)
	fmt.Printf("  终态: %s, 能否再打开: %v\n", connFSM.Current(), connFSM.Can(connfsm.Open))

	// 享元模式示例（字符串驻留 — 高性能去重）
	fmt.Println("\n--- 享元模式（字符串驻留） ---")
	intPool := intern.New()
	words := []string{"error", "warning", "error", "info", "error", "warning", "debug", "info"}
	for _, w := range words {
		intPool.Get(string([]byte(w)))
	}
	fmt.Printf("  输入 %d 个字符串, 驻留 %d 个唯一值\n", len(words), intPool.Len())
	fmt.Printf("  命中: %d, 未命中: %d\n", intPool.Hits(), intPool.Misses())

	// 享元模式示例（图标工厂 — 共享位图数据）
	fmt.Println("\n--- 享元模式（图标工厂） ---")
	iconFactory := icon.NewFactory()
	loader := func(name string) []byte { return []byte("bitmap:" + name) }
	i1 := iconFactory.Get("star", loader)
	i2 := iconFactory.Get("star", loader)
	i3 := iconFactory.Get("heart", loader)
	fmt.Printf("  star 同一指针: %v, 工厂缓存: %d 个图标\n", i1 == i2, iconFactory.Len())
	fmt.Printf("  star=%q, heart=%q\n", i1.Data, i3.Data)

	// 享元模式示例（样式注册表 + 颜色调色板）
	fmt.Println("\n--- 享元模式（样式注册表 & 颜色调色板） ---")
	styleReg := style.NewRegistry()
	st1 := styleReg.Get("Arial", "red", 14, false)
	st2 := styleReg.Get("Arial", "red", 14, false)
	st3 := styleReg.Get("Arial", "blue", 14, true)
	fmt.Printf("  同参数共享: %v, 不同参数独立: %v, 注册表: %d 种样式\n",
		st1 == st2, st1 == st3, styleReg.Len())

	palette := color.NewPalette()
	red := palette.Get("red", 255, 0, 0, 255)
	red2 := palette.Get("red", 255, 0, 0, 255)
	blue := palette.Get("blue", 0, 0, 255, 255)
	fmt.Printf("  red 共享: %v, 调色板: %d 种颜色\n", red == red2, palette.Len())
	_ = blue

	// 策略模式示例（退避策略：函数类型方式）
	fmt.Println("--- 策略模式（退避策略 — 函数类型） ---")
	attempt := 0
	err = retry.Retry(context.Background(), 3, retry.Exponential(10*time.Millisecond, 1*time.Second), func() error {
		attempt++
		fmt.Printf("  第 %d 次尝试...\n", attempt)
		if attempt < 3 {
			return errors.New("transient error")
		}
		return nil
	})
	if err != nil {
		fmt.Printf("重试失败: %v\n", err)
	} else {
		fmt.Printf("重试成功，共尝试 %d 次\n", attempt)
	}

	// 管道模式示例（channel 管道：Generate → Filter → Map → Collect）
	fmt.Println("\n--- 管道模式（channel 管道：Generate → Filter → Map → Collect） ---")
	{
		ctx := context.Background()
		in := stage.Generate(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
		filtered := stage.Filter(ctx, in, func(v int) bool { return v%2 == 0 })
		mapped := stage.Map(ctx, filtered, func(v int) int { return v * 10 })
		result := stage.Collect(mapped)
		fmt.Printf("偶数×10: %v\n", result)
	}

	// 管道模式示例（Fan-Out/Fan-In：4 个 worker 并行处理）
	fmt.Println("\n--- 管道模式（Fan-Out/Fan-In：4 个 worker 并行） ---")
	{
		ctx := context.Background()
		in := stage.Generate(ctx, 1, 2, 3, 4, 5, 6, 7, 8)
		workers := stage.FanOut(ctx, in, 4, func(v int) int { return v * v })
		merged := stage.Collect(stage.Merge(ctx, workers...))
		sort.Ints(merged)
		fmt.Printf("4 worker 平方: %v\n", merged)
	}

	// 管道模式示例（errgroup 生命周期管理）
	fmt.Println("\n--- 管道模式（errgroup 生命周期管理） ---")
	{
		raw := make(chan int)
		doubled := make(chan string)

		p := runner.New()
		p.Add(func(ctx context.Context) error {
			defer close(raw)
			for i := 1; i <= 5; i++ {
				select {
				case raw <- i:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
		p.Add(func(ctx context.Context) error {
			defer close(doubled)
			for v := range raw {
				select {
				case doubled <- fmt.Sprintf("  %d→%d", v, v*2):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
		p.Add(func(ctx context.Context) error {
			for s := range doubled {
				fmt.Println(s)
			}
			return nil
		})

		if pipeErr := p.Run(context.Background()); pipeErr != nil {
			fmt.Printf("pipeline err: %v\n", pipeErr)
		}
		fmt.Println("管道正常退出，所有 goroutine 已清理")
	}

	// 扇出扇入模式示例（Scatter: 有界并发映射，保序返回）
	fmt.Println("\n--- 扇出扇入模式（Scatter: 有界并发 3 worker） ---")
	{
		sites := []string{"go.dev", "github.com", "google.com", "golang.org", "pkg.go.dev"}
		results, scatterErr := parallel.Scatter(context.Background(), sites, 3,
			func(ctx context.Context, site string) (string, error) {
				time.Sleep(10 * time.Millisecond)
				return fmt.Sprintf("%s → 200 OK", site), nil
			},
		)
		if scatterErr != nil {
			fmt.Printf("scatter err: %v\n", scatterErr)
		} else {
			for _, r := range results {
				fmt.Printf("  %s\n", r)
			}
			fmt.Println("保序: 结果顺序与输入一致")
		}
	}

	// 扇出扇入模式示例（MapReduce: 并行统计 + 聚合）
	fmt.Println("\n--- 扇出扇入模式（MapReduce: 并行词数统计） ---")
	{
		texts := []string{"hello world", "go is great", "hello go world"}
		total, mrErr := mapreduce.MapReduce(context.Background(), texts, 2,
			func(_ context.Context, text string) (int, error) {
				return len(strings.Fields(text)), nil
			},
			func(acc, count int) int { return acc + count },
			0,
		)
		if mrErr != nil {
			fmt.Printf("mapreduce err: %v\n", mrErr)
		} else {
			fmt.Printf("文本: %v\n", texts)
			fmt.Printf("总词数: %d\n", total)
		}
	}

	// 扇出扇入模式示例（Race: 多源竞速，最快响应获胜）
	fmt.Println("\n--- 扇出扇入模式（Race: 最快响应获胜） ---")
	{
		winner, raceErr := race.First(context.Background(),
			func(ctx context.Context) (string, error) {
				time.Sleep(50 * time.Millisecond)
				return "mirror-A", nil
			},
			func(ctx context.Context) (string, error) {
				time.Sleep(5 * time.Millisecond)
				return "mirror-B", nil
			},
			func(ctx context.Context) (string, error) {
				time.Sleep(30 * time.Millisecond)
				return "mirror-C", nil
			},
		)
		if raceErr != nil {
			fmt.Printf("race err: %v\n", raceErr)
		} else {
			fmt.Printf("最先返回: %s（其余已取消）\n", winner)
		}
	}

	// 命令模式示例（文本编辑器：撤销/重做）
	fmt.Println("\n--- 命令模式（文本编辑器：撤销/重做） ---")
	{
		ed := editor.New()
		ed.Insert(0, "Hello World")
		fmt.Printf("  插入后: %q\n", ed.Content())

		ed.Delete(5, 6)
		fmt.Printf("  删除后: %q\n", ed.Content())

		ed.Insert(5, " Go")
		fmt.Printf("  再插入: %q\n", ed.Content())

		ed.Undo()
		fmt.Printf("  撤销1: %q\n", ed.Content())

		ed.Undo()
		fmt.Printf("  撤销2: %q\n", ed.Content())

		ed.Redo()
		fmt.Printf("  重做:   %q\n", ed.Content())

		fmt.Printf("  历史: %v\n", ed.History())
		fmt.Printf("  undo=%d redo=%d\n", ed.UndoCount(), ed.RedoCount())
	}

	// 命令模式示例（事务：批量操作 + 自动回滚）
	fmt.Println("\n--- 命令模式（事务：批量操作 + 自动回滚） ---")
	{
		store := transaction.NewStore()
		store.Put("name", "Alice")
		store.Put("balance", "100")
		fmt.Printf("  初始: name=%q balance=%q\n", mustGet(store, "name"), mustGet(store, "balance"))

		tx := transaction.Begin(
			transaction.Set(store, "name", "Bob"),
			transaction.Set(store, "balance", "200"),
			transaction.Del(store, "nonexistent"),
		)
		txErr := tx.Commit()
		fmt.Printf("  提交失败: %v\n", txErr)
		fmt.Printf("  回滚后: name=%q balance=%q（恢复原值）\n",
			mustGet(store, "name"), mustGet(store, "balance"))

		tx2 := transaction.Begin(
			transaction.Set(store, "name", "Bob"),
			transaction.Set(store, "balance", "200"),
		)
		tx2.Commit()
		fmt.Printf("  成功提交: name=%q balance=%q\n",
			mustGet(store, "name"), mustGet(store, "balance"))

		tx2.Rollback()
		fmt.Printf("  显式回滚: name=%q balance=%q\n",
			mustGet(store, "name"), mustGet(store, "balance"))
	}

	// 组合模式示例（文件树：叶子与组合统一接口）
	fmt.Println("\n--- 组合模式（文件树：统一接口递归聚合） ---")
	{
		src := fstree.NewDirectory("src")
		src.Add(
			fstree.NewFile("main.go", 1200),
			fstree.NewFile("handler.go", 800),
		)
		root := fstree.NewDirectory("project")
		root.Add(src, fstree.NewFile("README.md", 2048), fstree.NewFile("go.mod", 150))

		fstree.PrintTree(os.Stdout, root)
		fmt.Printf("总大小: %d bytes\n", root.Size())

		var fileCount int
		fstree.Walk(root, func(c fstree.Component) {
			if _, isDir := c.(*fstree.Directory); !isDir {
				fileCount++
			}
		})
		fmt.Printf("文件数: %d\n", fileCount)
	}

	// 组合模式示例（校验器组合：And/Or/Not 校验树）
	fmt.Println("\n--- 组合模式（校验器组合：And/Or/Not 校验树） ---")
	{
		usernameRule := validator.And(
			validator.NonEmpty(),
			validator.MinLength(3),
			validator.MaxLength(20),
			validator.Matches(`^[a-zA-Z0-9]+$`),
		)
		for _, name := range []string{"", "ab", "alice", "alice_bob"} {
			if vErr := usernameRule.Validate(name); vErr != nil {
				fmt.Printf("  %q → ✗ %v\n", name, vErr)
			} else {
				fmt.Printf("  %q → ✓\n", name)
			}
		}
	}
}

func mustGet(s *transaction.Store, key string) string {
	v, _ := s.Get(key)
	return v
}
