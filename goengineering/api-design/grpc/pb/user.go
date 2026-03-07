// Package pb 提供手写的 gRPC 消息类型，等价于 protoc 生成的代码。
//
// 在生产环境中应使用 .proto 文件 + protoc 生成代码。
// 本示例手写类型以避免工具链依赖，专注于展示 gRPC 设计模式。
package pb

import "time"

// User 是用户资源消息。
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int32     `json:"age"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserRequest 是创建用户的请求消息。
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int32  `json:"age"`
}

// GetUserRequest 是获取用户的请求消息。
type GetUserRequest struct {
	ID string `json:"id"`
}

// ListUsersRequest 是列出用户的请求消息。
type ListUsersRequest struct {
	PageSize  int32  `json:"page_size"`
	PageToken string `json:"page_token"`
}

// ListUsersResponse 是用户列表响应消息。
type ListUsersResponse struct {
	Users         []*User `json:"users"`
	NextPageToken string  `json:"next_page_token"`
}

// DeleteUserRequest 是删除用户的请求消息。
type DeleteUserRequest struct {
	ID string `json:"id"`
}

// Empty 等价于 google.protobuf.Empty。
type Empty struct{}
