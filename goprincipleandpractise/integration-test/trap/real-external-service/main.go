package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"time"
)

/*
陷阱：测试直接依赖外部真实服务

运行：go run .

预期行为：
  1. 模拟调用"外部 API"（随机超时），展示测试因网络波动而 Flaky
  2. 启动本地 Mock Server 替代外部 API，展示测试稳定通过

  正确做法：外部 API 用 httptest.NewServer 模拟，数据库用 testcontainers。
*/

// PaymentClient 模拟一个依赖外部支付 API 的客户端
type PaymentClient struct {
	baseURL string
	client  *http.Client
}

func NewPaymentClient(baseURL string) *PaymentClient {
	return &PaymentClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 100 * time.Millisecond}, // 超时设得很短
	}
}

func (c *PaymentClient) Pay(amount int) (string, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/pay?amount=%d", c.baseURL, amount))
	if err != nil {
		return "", fmt.Errorf("payment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("payment failed with status %d", resp.StatusCode)
	}

	var result struct {
		TradeNo string `json:"trade_no"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.TradeNo, nil
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// ========== 模拟不稳定的外部 API ==========
	unstableAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟外部 API 的不稳定性：50% 概率延迟超过客户端超时
		latency := time.Duration(rand.Intn(200)) * time.Millisecond
		time.Sleep(latency)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"trade_no": fmt.Sprintf("TXN_%d", time.Now().UnixNano()),
		})
	}))
	defer unstableAPI.Close()

	fmt.Println("=== ❌ 错误做法：依赖不稳定的外部 API ===")
	fmt.Println()

	client := NewPaymentClient(unstableAPI.URL)
	passed, failed := 0, 0
	for i := 1; i <= 10; i++ {
		tradeNo, err := client.Pay(100)
		if err != nil {
			fmt.Printf("  第%2d次: FAIL — %v\n", i, err)
			failed++
		} else {
			fmt.Printf("  第%2d次: PASS — trade_no=%s\n", i, tradeNo)
			passed++
		}
	}
	fmt.Printf("\n  结果: %d 通过, %d 失败 (通过率 %.0f%%)\n",
		passed, failed, float64(passed)/10*100)
	fmt.Println("  问题: 外部 API 延迟不可控，导致测试时好时坏")
	fmt.Println("  更严重: 如果是付费 API，每次 CI 运行都在烧钱")

	fmt.Println()

	// ========== 正确做法：启动本地 Mock Server ==========
	stableAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock Server：固定延迟 1ms，永远返回成功
		time.Sleep(1 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"trade_no": fmt.Sprintf("MOCK_TXN_%d", time.Now().UnixNano()),
		})
	}))
	defer stableAPI.Close()

	fmt.Println("=== ✅ 正确做法：使用本地 Mock Server ===")
	fmt.Println()

	mockClient := NewPaymentClient(stableAPI.URL)
	passed = 0
	for i := 1; i <= 10; i++ {
		tradeNo, err := mockClient.Pay(100)
		if err != nil {
			fmt.Printf("  第%2d次: FAIL — %v\n", i, err)
		} else {
			fmt.Printf("  第%2d次: PASS — trade_no=%s\n", i, tradeNo)
			passed++
		}
	}
	fmt.Printf("\n  结果: %d/10 通过 (通过率 100%%)\n", passed)
	fmt.Println("  优点:")
	fmt.Println("    1. 延迟可控，测试结果确定性")
	fmt.Println("    2. 不消耗真实资源，不产生费用")
	fmt.Println("    3. 可以模拟各种错误场景（超时、500、限流）")
	fmt.Println("    4. 离线也能跑测试")
}
