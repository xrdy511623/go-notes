package apidesigngrpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "go-notes/goprincipleandpractise/api-design/grpc/pb"
)

func TestUserServiceCRUD(t *testing.T) {
	svc := NewUserService()
	ctx := context.Background()

	// ── Create ───────────────────────────────────
	user, err := svc.CreateUser(ctx, &pb.CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Age:   30,
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if user.ID == "" {
		t.Fatal("CreateUser: empty ID")
	}
	if user.Name != "Alice" {
		t.Fatalf("CreateUser: name = %q, want Alice", user.Name)
	}

	// ── Get ──────────────────────────────────────
	got, err := svc.GetUser(ctx, &pb.GetUserRequest{ID: user.ID})
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Email != "alice@example.com" {
		t.Fatalf("GetUser: email = %q, want alice@example.com", got.Email)
	}

	// ── List ─────────────────────────────────────
	list, err := svc.ListUsers(ctx, &pb.ListUsersRequest{PageSize: 10})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(list.Users) != 1 {
		t.Fatalf("ListUsers: got %d users, want 1", len(list.Users))
	}

	// ── Delete ───────────────────────────────────
	_, err = svc.DeleteUser(ctx, &pb.DeleteUserRequest{ID: user.ID})
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	// ── Get after delete → NotFound ──────────────
	_, err = svc.GetUser(ctx, &pb.GetUserRequest{ID: user.ID})
	if err == nil {
		t.Fatal("GetUser after delete: expected error, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.NotFound {
		t.Fatalf("GetUser after delete: expected NotFound, got %v", err)
	}
}

func TestUserServiceErrors(t *testing.T) {
	svc := NewUserService()
	ctx := context.Background()

	tests := []struct {
		name     string
		fn       func() error
		wantCode codes.Code
	}{
		{
			name: "create: missing name",
			fn: func() error {
				_, err := svc.CreateUser(ctx, &pb.CreateUserRequest{Email: "a@b.com"})
				return err
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "create: missing email",
			fn: func() error {
				_, err := svc.CreateUser(ctx, &pb.CreateUserRequest{Name: "Bob"})
				return err
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "get: missing id",
			fn: func() error {
				_, err := svc.GetUser(ctx, &pb.GetUserRequest{})
				return err
			},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "get: not found",
			fn: func() error {
				_, err := svc.GetUser(ctx, &pb.GetUserRequest{ID: "nonexistent"})
				return err
			},
			wantCode: codes.NotFound,
		},
		{
			name: "delete: not found",
			fn: func() error {
				_, err := svc.DeleteUser(ctx, &pb.DeleteUserRequest{ID: "nonexistent"})
				return err
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("code = %v, want %v", st.Code(), tt.wantCode)
			}
		})
	}
}

func TestDuplicateEmail(t *testing.T) {
	svc := NewUserService()
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, &pb.CreateUserRequest{
		Name: "Alice", Email: "dup@example.com",
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.CreateUser(ctx, &pb.CreateUserRequest{
		Name: "Bob", Email: "dup@example.com",
	})
	if err == nil {
		t.Fatal("expected AlreadyExists error, got nil")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.AlreadyExists {
		t.Errorf("code = %v, want AlreadyExists", st.Code())
	}
}

func TestGRPCCodeToHTTP(t *testing.T) {
	tests := []struct {
		code     codes.Code
		wantHTTP int
	}{
		{codes.OK, 200},
		{codes.InvalidArgument, 400},
		{codes.Unauthenticated, 401},
		{codes.PermissionDenied, 403},
		{codes.NotFound, 404},
		{codes.AlreadyExists, 409},
		{codes.ResourceExhausted, 429},
		{codes.Internal, 500},
		{codes.Unavailable, 503},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			got := GRPCCodeToHTTP(tt.code)
			if got != tt.wantHTTP {
				t.Errorf("GRPCCodeToHTTP(%v) = %d, want %d", tt.code, got, tt.wantHTTP)
			}
		})
	}
}
