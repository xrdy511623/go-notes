package pattern

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFetchAllBasicSuccess(t *testing.T) {
	tasks := []Task{
		{ID: "a", Data: 1},
		{ID: "b", Data: 2},
		{ID: "c", Data: 3},
	}

	err := FetchAllBasic(context.Background(), tasks, func(ctx context.Context, t Task) error {
		return nil // 全部成功
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchAllBasicFirstError(t *testing.T) {
	tasks := []Task{
		{ID: "a", Data: 1},
		{ID: "b", Data: 2},
	}

	sentinel := errors.New("task failed")
	err := FetchAllBasic(context.Background(), tasks, func(ctx context.Context, t Task) error {
		if t.ID == "b" {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got: %v", err)
	}
}

func TestFetchAllWithLimit(t *testing.T) {
	tasks := make([]Task, 20)
	for i := range tasks {
		tasks[i] = Task{ID: string(rune('a' + i)), Data: i}
	}

	err := FetchAllWithLimit(context.Background(), tasks, 5, func(ctx context.Context, t Task) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchAllCollectErrors(t *testing.T) {
	tasks := []Task{
		{ID: "a", Data: 1},
		{ID: "b", Data: 2},
		{ID: "c", Data: 3},
	}

	err := FetchAllCollectErrors(context.Background(), tasks, func(ctx context.Context, t Task) error {
		if t.Data%2 == 0 {
			return errors.New("even data not allowed")
		}
		return nil
	})

	if err == nil {
		t.Fatal("expected error for even data")
	}

	// 只有task b失败，应该只有一个错误
	errStr := err.Error()
	if !strings.Contains(errStr, "task b") {
		t.Errorf("error should mention task b, got: %s", errStr)
	}
}
