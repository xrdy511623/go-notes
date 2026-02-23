package mockgendemo

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
)

// ---------- 表驱动测试 + gomock ----------

func TestUserService_Register(t *testing.T) {
	tests := []struct {
		name      string
		userName  string
		email     string
		setupMock func(store *MockUserStore)
		wantErr   error
		wantUser  bool
	}{
		{
			name:     "成功注册新用户",
			userName: "alice",
			email:    "alice@example.com",
			setupMock: func(store *MockUserStore) {
				store.EXPECT().
					GetByEmail(gomock.Any(), "alice@example.com").
					Return(nil, ErrNotFound).
					Times(1)
				store.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil).
					Times(1)
			},
			wantErr:  nil,
			wantUser: true,
		},
		{
			name:     "邮箱已存在",
			userName: "alice",
			email:    "alice@example.com",
			setupMock: func(store *MockUserStore) {
				store.EXPECT().
					GetByEmail(gomock.Any(), "alice@example.com").
					Return(&User{ID: "existing", Email: "alice@example.com"}, nil).
					Times(1)
				// Create不应被调用（不设置期望）
			},
			wantErr:  ErrAlreadyExists,
			wantUser: false,
		},
		{
			name:     "空邮箱",
			userName: "alice",
			email:    "",
			setupMock: func(store *MockUserStore) {
				// 不应调用任何store方法
			},
			wantErr:  ErrInvalidEmail,
			wantUser: false,
		},
		{
			name:     "存储层错误",
			userName: "alice",
			email:    "alice@example.com",
			setupMock: func(store *MockUserStore) {
				store.EXPECT().
					GetByEmail(gomock.Any(), "alice@example.com").
					Return(nil, errors.New("db connection lost")).
					Times(1)
			},
			wantErr:  nil, // 非nil，但不是特定的sentinel error
			wantUser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			store := NewMockUserStore(ctrl)
			tt.setupMock(store)

			svc := NewUserService(store)
			user, err := svc.Register(context.Background(), tt.userName, tt.email)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("err = %v, want %v", err, tt.wantErr)
				}
			} else if tt.name == "存储层错误" {
				if err == nil {
					t.Error("expected non-nil error for db failure")
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantUser && user == nil {
				t.Error("expected non-nil user")
			}
			if !tt.wantUser && user != nil {
				t.Errorf("expected nil user, got %+v", user)
			}
		})
	}
}

// TestUserService_Register_CallOrder 演示gomock的调用顺序断言
func TestUserService_Register_CallOrder(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := NewMockUserStore(ctrl)

	// InOrder确保GetByEmail在Create之前被调用
	first := store.EXPECT().
		GetByEmail(gomock.Any(), "bob@example.com").
		Return(nil, ErrNotFound)
	store.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil).
		After(first)

	svc := NewUserService(store)
	user, err := svc.Register(context.Background(), "bob", "bob@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "bob" {
		t.Errorf("user.Name = %q, want %q", user.Name, "bob")
	}
}
