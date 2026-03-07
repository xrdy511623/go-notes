package apidesigngrpc

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ── 拦截器（Interceptor）────────────────────────────

// LoggingInterceptor 记录每个 gRPC 调用的方法名、耗时和错误。
func LoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	code := status.Code(err)
	log.Printf("[gRPC] %s → %s (%s)", info.FullMethod, code, time.Since(start))
	return resp, err
}

// AuthInterceptor 验证 gRPC metadata 中的 authorization token。
func AuthInterceptor(validToken string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		if tokens[0] != "Bearer "+validToken {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		return handler(ctx, req)
	}
}

// RecoveryInterceptor 捕获 handler panic，转换为 Internal 错误。
func RecoveryInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[gRPC PANIC] %s: %v", info.FullMethod, r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

// NewGRPCServer 创建配置好拦截器链的 gRPC 服务器。
//
// 拦截器执行顺序: Recovery → Logging → Auth
func NewGRPCServer(authToken string) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			RecoveryInterceptor,
			LoggingInterceptor,
			AuthInterceptor(authToken),
		),
	)
}
