package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
)

func main() {
	trapFatalInGoroutine()
	trapBadKeyValue()
	trapLogAndSwallow()
	trapSensitiveData()
}

// ============================================================
// 陷阱1：log.Fatal在goroutine中
// log.Fatal调用os.Exit(1)，不执行defer，不等待其他goroutine
// ============================================================

func trapFatalInGoroutine() {
	fmt.Println("=== 陷阱1：log.Fatal在goroutine中 ===")

	// 危险示范（注释掉，因为会直接退出进程）
	// go func() {
	//     log.Fatal("fatal in goroutine") // 整个进程直接退出！
	// }()

	// 正确做法：goroutine中用slog.Error + 返回错误
	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := fmt.Errorf("simulated error")
		slog.Error("operation failed", "error", err)
		errCh <- err
	}()

	wg.Wait()
	close(errCh)
	if err := <-errCh; err != nil {
		fmt.Println("正确：主goroutine收到错误并决定是否退出:", err)
	}
	fmt.Println()
}

// ============================================================
// 陷阱2：slog交替key-value奇数参数
// key-value不成对时不会panic，但会输出 !BADKEY= 标记
// ============================================================

func trapBadKeyValue() {
	fmt.Println("=== 陷阱2：slog key-value奇数参数 ===")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 移除time字段以简化输出
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))

	// 错误：3个参数（"attempt"没有对应的value）
	fmt.Println("错误写法（奇数参数）:")
	logger.Info("user login", "username", "alice", "attempt")
	// 输出: level=INFO msg="user login" username=alice !BADKEY=attempt

	// 正确：key-value成对
	fmt.Println("正确写法（偶数参数）:")
	logger.Info("user login", "username", "alice", "attempt", 3)
	fmt.Println()
}

// ============================================================
// 陷阱3：记录了日志但吞掉了error
// 调用方以为成功了，实际已经失败
// ============================================================

func riskyOperation() error {
	return fmt.Errorf("connection refused")
}

// 错误：记了日志但没return err
func doSomethingWrong() error {
	if err := riskyOperation(); err != nil {
		log.Printf("operation failed: %v", err)
		// 忘记 return err！
	}
	return nil // 调用方以为成功了
}

// 正确：记日志并返回error
func doSomethingRight() error {
	if err := riskyOperation(); err != nil {
		return fmt.Errorf("do something: %w", err) // 底层只wrap，不记日志
	}
	return nil
}

func trapLogAndSwallow() {
	fmt.Println("=== 陷阱3：记日志但吞掉error ===")

	err := doSomethingWrong()
	fmt.Println("错误写法 — 调用方收到的err:", err) // nil！

	err = doSomethingRight()
	fmt.Println("正确写法 — 调用方收到的err:", err) // 有错误
	fmt.Println()
}

// ============================================================
// 陷阱4：日志中输出敏感信息
// ============================================================

func trapSensitiveData() {
	fmt.Println("=== 陷阱4：日志中输出敏感信息 ===")

	type LoginRequest struct {
		Username string
		Password string
		Token    string
	}

	req := LoginRequest{
		Username: "alice",
		Password: "s3cret!",
		Token:    "eyJhbGciOiJIUzI1NiJ9...",
	}

	fmt.Println("错误写法 — 直接打印整个struct（含密码和token）:")
	fmt.Printf("  log.Printf(\"login: %%+v\", req) → %+v\n", req)

	fmt.Println("正确写法 — 只记录安全字段:")
	slog.Info("user login",
		"username", req.Username,
		"has_token", req.Token != "",
		// 不记录password和token
	)
	fmt.Println()
}
