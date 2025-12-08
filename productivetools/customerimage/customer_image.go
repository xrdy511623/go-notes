package customerimage

import (
	"context"

	"tcg-uss-ae-go-sp/internal/apiserver/persistence/cache"
	"tcg-uss-ae-go-sp/internal/apiserver/persistence/repository"
	"tcg-uss-ae-go-sp/internal/apiserver/types"
	"tcg-uss-ae-go-sp/internal/pkg/kafka"
)

// Service defines country related query operations.
type Service interface {
	GetCustomerImageByID(ctx context.Context, customerId int64) (types.ApiBaseValueResp, error)
}

type service struct {
	customerImageRepo      repository.CustomerImage
	customerProfileRepo    repository.CustomerProfile
	customerImmutableCache cache.CustomerImmutableCache
	kafkaClient            *kafka.Client
}

// NewService constructs a Service backed by the provided repository.
func NewService(customerImageRepo repository.CustomerImage, customerProfileRepo repository.CustomerProfile, customerImmutableCache cache.CustomerImmutableCache, kafkaClient *kafka.Client) Service {
	return &service{
		customerImageRepo:      customerImageRepo,
		customerProfileRepo:    customerProfileRepo,
		customerImmutableCache: customerImmutableCache,
		kafkaClient:            kafkaClient,
	}
}
