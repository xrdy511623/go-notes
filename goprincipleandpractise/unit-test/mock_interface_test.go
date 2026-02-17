package unittest

import (
	"errors"
	"testing"
)

// ---------- 接口 Mock 测试 ----------
//
// 本文件演示"手写 Stub"方式测试 UserService。
//
// 三种 Mock 方式对比：
//
// | 方式       | 依赖         | 优点                 | 缺点                     |
// |-----------|-------------|---------------------|-------------------------|
// | 手写 Stub  | 无           | 零依赖、灵活、易读     | 接口方法多时代码量大        |
// | gomock    | mockgen 生成 | 自动生成、严格调用验证  | 需要代码生成步骤           |
// | gomonkey  | runtime patch| 不依赖接口           | 不支持 -race、平台限制     |
//
// 推荐优先使用手写 Stub，接口方法 ≤5 个时完全够用。

func TestUserService_GetUser(t *testing.T) {
	tests := []struct {
		name    string
		store   *StubUserStore
		userID  string
		want    *User
		wantErr error
	}{
		{
			name: "成功获取用户",
			store: &StubUserStore{
				GetByIDFunc: func(id string) (*User, error) {
					return &User{ID: "1", Name: "Alice", Email: "alice@example.com"}, nil
				},
			},
			userID: "1",
			want:   &User{ID: "1", Name: "Alice", Email: "alice@example.com"},
		},
		{
			name: "用户不存在",
			store: &StubUserStore{
				GetByIDFunc: func(id string) (*User, error) {
					return nil, nil // 返回 nil 表示未找到
				},
			},
			userID:  "999",
			wantErr: ErrUserNotFound,
		},
		{
			name: "存储层错误",
			store: &StubUserStore{
				GetByIDFunc: func(id string) (*User, error) {
					return nil, errors.New("connection refused")
				},
			},
			userID:  "1",
			wantErr: errors.New("connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(tt.store)
			got, err := svc.GetUser(tt.userID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					// 允许 wrapped error 匹配
					if !containsError(err, tt.wantErr) {
						t.Fatalf("expected error containing %q, got %q", tt.wantErr, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.want.ID || got.Name != tt.want.Name || got.Email != tt.want.Email {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestUserService_CreateUser(t *testing.T) {
	tests := []struct {
		name    string
		store   *StubUserStore
		inName  string
		inEmail string
		wantErr error
	}{
		{
			name: "成功创建",
			store: &StubUserStore{
				SaveFunc: func(user *User) error { return nil },
			},
			inName:  "Bob",
			inEmail: "bob@example.com",
		},
		{
			name:    "名字为空",
			store:   &StubUserStore{},
			inName:  "",
			inEmail: "bob@example.com",
			wantErr: ErrEmptyName,
		},
		{
			name:    "邮箱为空",
			store:   &StubUserStore{},
			inName:  "Bob",
			inEmail: "",
			wantErr: ErrEmptyEmail,
		},
		{
			name: "保存失败",
			store: &StubUserStore{
				SaveFunc: func(user *User) error { return errors.New("disk full") },
			},
			inName:  "Bob",
			inEmail: "bob@example.com",
			wantErr: errors.New("disk full"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(tt.store)
			_, err := svc.CreateUser(tt.inName, tt.inEmail)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) && !containsError(err, tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// containsError 检查 err 的错误链或消息中是否包含 target
func containsError(err, target error) bool {
	if errors.Is(err, target) {
		return true
	}
	// 对于非 sentinel error，检查消息是否包含
	return err != nil && target != nil &&
		len(err.Error()) >= len(target.Error()) &&
		contains(err.Error(), target.Error())
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
