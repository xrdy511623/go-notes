package customerimage

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
	"tcg-uss-ae-go-sp/internal/apiserver/model"
	"tcg-uss-ae-go-sp/internal/apiserver/persistence/cache"
	"tcg-uss-ae-go-sp/internal/apiserver/types"
	"tcg-uss-ae-go-sp/internal/pkg/code"
	"tcg-uss-ae-go-sp/internal/pkg/lib"
)

// stubImmutableCache fakes the immutable cache with configurable return values.
type stubImmutableCache struct {
	resultByID *cache.CustomerImmutableTO
	errByID    error
	called     bool
}

func (s *stubImmutableCache) GetById(ctx context.Context, customerId int64) (*cache.CustomerImmutableTO, error) {
	s.called = true
	return s.resultByID, s.errByID
}

// Unused interface methods for this test.
func (s *stubImmutableCache) GetByName(ctx context.Context, customerName string) (*cache.CustomerImmutableTO, error) {
	return nil, nil
}
func (s *stubImmutableCache) RemoveById(ctx context.Context, customerId int64) error { return nil }
func (s *stubImmutableCache) RemoveAll()                                             {}

// stubCustomerImageRepo fakes the repository.
type stubCustomerImageRepo struct {
	result model.CustomerImage
	err    error
	called bool
}

func (s *stubCustomerImageRepo) Find(ctx context.Context, customerId int64) (model.CustomerImage, error) {
	s.called = true
	return s.result, s.err
}

// Unused interface methods for this test.
func (s *stubCustomerImageRepo) Insert(ctx context.Context, ci model.CustomerImage) error { return nil }
func (s *stubCustomerImageRepo) Update(ctx context.Context, ci model.CustomerImage) error { return nil }
func (s *stubCustomerImageRepo) UpdateAvatar(ctx context.Context, customerId int64, avatar string) error {
	return nil
}
func (s *stubCustomerImageRepo) Upsert(ctx context.Context, ci model.CustomerImage) error { return nil }
func (s *stubCustomerImageRepo) Delete(ctx context.Context, customerId int64) error       { return nil }

func TestGetCustomerImageByIDSuccess(t *testing.T) {
	cacheStub := &stubImmutableCache{
		resultByID: &cache.CustomerImmutableTO{CustomerID: 42},
	}
	repoStub := &stubCustomerImageRepo{
		result: model.CustomerImage{
			CustomerID:    42,
			IDCardFront:   sql.NullString{String: "front.png", Valid: true},
			IDCardBack:    sql.NullString{String: "back.png", Valid: true},
			Avatar:        sql.NullString{String: "avatar.png", Valid: true},
			ExtraImageOne: sql.NullString{String: "extra1.png", Valid: true},
		},
	}

	svc := &service{customerImageRepo: repoStub, customerImmutableCache: cacheStub}
	resp, err := svc.GetCustomerImageByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !cacheStub.called {
		t.Fatalf("expected immutable cache to be called")
	}
	if !repoStub.called {
		t.Fatalf("expected repository to be called")
	}

	img, ok := resp.Value.(types.CustomerImage)
	if !ok {
		t.Fatalf("response type mismatch: %T", resp.Value)
	}
	if img.CustomerID != 42 {
		t.Fatalf("unexpected CustomerID: %d", img.CustomerID)
	}
	assertStringValue(t, img.IDCardFront, "front.png")
	assertStringValue(t, img.IDCardBack, "back.png")
	assertStringValue(t, img.Avatar, "avatar.png")
	assertStringValue(t, img.ExtraImageOne, "extra1.png")
}

func TestGetCustomerImageByIDSetsIDWhenMissing(t *testing.T) {
	cacheStub := &stubImmutableCache{resultByID: &cache.CustomerImmutableTO{CustomerID: 99}}
	repoStub := &stubCustomerImageRepo{
		// CustomerID is intentionally zero to exercise the fallback.
		result: model.CustomerImage{},
	}

	svc := &service{customerImageRepo: repoStub, customerImmutableCache: cacheStub}
	resp, err := svc.GetCustomerImageByID(context.Background(), 99)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	img := resp.Value.(types.CustomerImage)
	if img.CustomerID != 99 {
		t.Fatalf("expected CustomerID to default to requested ID, got %d", img.CustomerID)
	}
}

func TestGetCustomerImageByIDCacheError(t *testing.T) {
	cacheStub := &stubImmutableCache{errByID: errors.New("cache down")}
	repoStub := &stubCustomerImageRepo{}

	svc := &service{customerImageRepo: repoStub, customerImmutableCache: cacheStub}
	_, err := svc.GetCustomerImageByID(context.Background(), 1)
	if err == nil || err.Error() != "cache down" {
		t.Fatalf("expected cache error, got %v", err)
	}
	if repoStub.called {
		t.Fatalf("repo should not be called when cache fails")
	}
}

func TestGetCustomerImageByIDCustomerMissing(t *testing.T) {
	cacheStub := &stubImmutableCache{resultByID: nil}
	repoStub := &stubCustomerImageRepo{}

	svc := &service{customerImageRepo: repoStub, customerImmutableCache: cacheStub}
	_, err := svc.GetCustomerImageByID(context.Background(), 123)
	var ue *lib.UssAeError
	if !errors.As(err, &ue) {
		t.Fatalf("expected UssAeError, got %v", err)
	}
	if ue.ErrorCode != code.DataNotFound {
		t.Fatalf("unexpected error code: %s", ue.ErrorCode)
	}
	if repoStub.called {
		t.Fatalf("repo should not be called when customer missing")
	}
}

func TestGetCustomerImageByIDRepoError(t *testing.T) {
	cacheStub := &stubImmutableCache{resultByID: &cache.CustomerImmutableTO{CustomerID: 7}}
	repoStub := &stubCustomerImageRepo{err: errors.New("repo failure")}

	svc := &service{customerImageRepo: repoStub, customerImmutableCache: cacheStub}
	_, err := svc.GetCustomerImageByID(context.Background(), 7)
	if err == nil || err.Error() != "repo failure" {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func assertStringValue(t *testing.T, val interface{}, want string) {
	t.Helper()
	v, ok := val.(*structpb.Value)
	if !ok {
		t.Fatalf("expected *structpb.Value, got %T", val)
	}
	if v.GetStringValue() != want {
		t.Fatalf("string value mismatch: got %q, want %q", v.GetStringValue(), want)
	}
}
