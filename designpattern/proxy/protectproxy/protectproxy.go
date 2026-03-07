package protectproxy

import (
	"context"
	"errors"

	"go-notes/designpattern/proxy/fetcher"
)

type ctxKey struct{}

// WithRole 在 context 中携带角色信息。
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, ctxKey{}, role)
}

// ErrUnauthorized 表示调用方没有访问权限。
var ErrUnauthorized = errors.New("protectproxy: unauthorized")

// Proxy 是保护代理——在转发请求前检查调用方的权限。
//
// 代理行为：
//   - 角色在允许列表中 → 转发给真实 Fetcher
//   - 角色不在允许列表中 → 返回 ErrUnauthorized，**不调用**真实 Fetcher
//
// 典型场景：API 网关的鉴权层、数据库访问控制、微服务的 sidecar proxy。
type Proxy struct {
	real         fetcher.Fetcher
	allowedRoles map[string]bool
}

// New 创建一个保护代理，只允许指定角色访问真实 Fetcher。
func New(real fetcher.Fetcher, roles ...string) *Proxy {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return &Proxy{real: real, allowedRoles: allowed}
}

func (p *Proxy) Fetch(ctx context.Context, key string) (string, error) {
	role, _ := ctx.Value(ctxKey{}).(string)
	if !p.allowedRoles[role] {
		return "", ErrUnauthorized
	}
	return p.real.Fetch(ctx, key)
}
