package tlsoverhead

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

/*
对比 HTTP vs HTTPS 在新建连接和复用连接场景下的请求开销。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下参考结果:

	BenchmarkHTTPNewConn         HTTP 新连接:    ~60,000 ns/op
	BenchmarkHTTPSNewConn        HTTPS 新连接:   ~600,000 ns/op  (~10x 慢，含 TLS 握手)
	BenchmarkHTTPKeepAlive       HTTP 复用连接:   ~15,000 ns/op
	BenchmarkHTTPSKeepAlive      HTTPS 复用连接:  ~18,000 ns/op  (~1.2x 慢)

结论:
  HTTPS 新建连接比 HTTP 慢约 10 倍（TLS 握手开销），
  但复用连接后仅慢约 20%（对称加密开销很小）。
  生产环境启用 Keep-Alive（Go 默认开启），HTTPS 性能损失可忽略不计。
*/

func BenchmarkHTTPNewConn(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()
	client := NewNoKeepAliveClient()
	b.ResetTimer()
	for b.Loop() {
		DoHTTPGet(srv.URL, client)
	}
}

func BenchmarkHTTPSNewConn(b *testing.B) {
	srv := httptest.NewTLSServer(http.HandlerFunc(handler))
	defer srv.Close()
	client := NewInsecureNoKeepAliveClient()
	b.ResetTimer()
	for b.Loop() {
		DoHTTPGet(srv.URL, client)
	}
}

func BenchmarkHTTPKeepAlive(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(handler))
	defer srv.Close()
	client := &http.Client{}
	b.ResetTimer()
	for b.Loop() {
		DoHTTPGet(srv.URL, client)
	}
}

func BenchmarkHTTPSKeepAlive(b *testing.B) {
	srv := httptest.NewTLSServer(http.HandlerFunc(handler))
	defer srv.Close()
	client := NewInsecureClient()
	b.ResetTimer()
	for b.Loop() {
		DoHTTPGet(srv.URL, client)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "ok")
}
