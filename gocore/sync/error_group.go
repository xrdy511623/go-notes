package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

var urls = []string{"http://www.sostupidname.com/", "http://www.golang.org/", "http://www.google.com/"}

func LimitGNum() {
	results := make(chan string, len(urls))
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// 用WithContext函数创建Group对象
	eg, ctx := errgroup.WithContext(timeoutCtx)
	// 调用SetLimit方法，设置可同时运行的最大协程数
	eg.SetLimit(3)
	for _, url := range urls {
		url := url
		// 调用Go方法
		eg.Go(func() error {
			// Fetch the URL.
			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return err
			}
			httpClient := http.Client{}
			resp, err := httpClient.Do(req)
			if err != nil {
				fmt.Printf("Failed to fetch %s\n", url)
				return err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			select {
			case results <- string(body):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}
	// Wait for all HTTP fetches to complete.
	// 等待所有任务执行完成，并对错误进行处理
	if err := eg.Wait(); err != nil {
		fmt.Println("Failed to fetched all URLs.")
	}
	close(results)
	// 打印结果
	for i := 0; i < len(urls); i++ {
		val := <-results
		fmt.Println(val)
	}
}

func main() {
	LimitGNum()
}
