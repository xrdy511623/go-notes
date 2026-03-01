// Package apidesigngrpc 演示 gRPC 服务实现和 Status 错误映射。
//
// 包名使用 apidesigngrpc 避免与 google.golang.org/grpc 冲突。
package apidesigngrpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "go-notes/goprincipleandpractise/api-design/grpc/pb"
)

// UserService 实现用户管理 gRPC 服务。
type UserService struct {
	mu    sync.RWMutex
	users map[string]*pb.User
	seq   int
}

// NewUserService 创建 UserService。
func NewUserService() *UserService {
	return &UserService{users: make(map[string]*pb.User)}
}

// CreateUser 创建新用户。
func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查 email 唯一性
	for _, u := range s.users {
		if u.Email == req.Email {
			return nil, status.Errorf(codes.AlreadyExists, "user with email %q already exists", req.Email)
		}
	}

	s.seq++
	user := &pb.User{
		ID:        fmt.Sprintf("usr_%06d", s.seq),
		Name:      req.Name,
		Email:     req.Email,
		Age:       req.Age,
		CreatedAt: time.Now().UTC(),
	}
	s.users[user.ID] = user
	return user, nil
}

// GetUser 获取用户详情。
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if req.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[req.ID]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "user %q not found", req.ID)
	}
	return user, nil
}

// ListUsers 分页列出用户。
func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	all := make([]*pb.User, 0, len(s.users))
	for _, u := range s.users {
		all = append(all, u)
	}

	// 简化分页: 用 offset 而非真正的 page token
	start := 0
	if req.PageToken != "" {
		for i, u := range all {
			if u.ID == req.PageToken {
				start = i + 1
				break
			}
		}
	}

	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}

	resp := &pb.ListUsersResponse{
		Users: all[start:end],
	}
	if end < len(all) {
		resp.NextPageToken = all[end-1].ID
	}
	return resp, nil
}

// DeleteUser 删除用户。
func (s *UserService) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.Empty, error) {
	if req.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[req.ID]; !ok {
		return nil, status.Errorf(codes.NotFound, "user %q not found", req.ID)
	}
	delete(s.users, req.ID)
	return &pb.Empty{}, nil
}

// GRPCCodeToHTTP 将 gRPC Status Code 映射到 HTTP 状态码。
// 展示两种协议间的映射关系。
func GRPCCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.OK:
		return 200
	case codes.InvalidArgument:
		return 400
	case codes.Unauthenticated:
		return 401
	case codes.PermissionDenied:
		return 403
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.FailedPrecondition:
		return 412
	case codes.ResourceExhausted:
		return 429
	case codes.Internal:
		return 500
	case codes.Unavailable:
		return 503
	default:
		return 500
	}
}
