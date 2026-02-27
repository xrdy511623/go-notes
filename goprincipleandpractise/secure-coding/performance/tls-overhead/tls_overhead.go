package tlsoverhead

import (
	"crypto/tls"
	"io"
	"net/http"
)

// DoHTTPGet 发送 HTTP GET 请求
func DoHTTPGet(url string, client *http.Client) (int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	return len(body), nil
}

// NewInsecureClient 创建跳过证书验证的 HTTPS 客户端（仅测试用）
func NewInsecureClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // 测试自签名证书
			},
		},
	}
}

// NewInsecureNoKeepAliveClient 创建不复用连接的 HTTPS 客户端（测试握手开销）
func NewInsecureNoKeepAliveClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // 测试自签名证书
			},
			DisableKeepAlives: true,
		},
	}
}

// NewNoKeepAliveClient 创建不复用连接的 HTTP 客户端
func NewNoKeepAliveClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
}
