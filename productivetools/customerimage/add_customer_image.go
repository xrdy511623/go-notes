package customerimage

import (
	"context"
	"database/sql"
	"fmt"

	"tcg-uss-ae-go-sp/internal/apiserver/model"
	"tcg-uss-ae-go-sp/internal/apiserver/types"
	"tcg-uss-ae-go-sp/internal/pkg/code"
	"tcg-uss-ae-go-sp/internal/pkg/consts"
	"tcg-uss-ae-go-sp/internal/pkg/lib"
)

func (s *service) AddCustomerImage(ctx context.Context, req types.CustomerImageTO) (types.ApiBaseValueResp, error) {
	customer, err := s.customerImmutableCache.GetById(ctx, req.CustomerId)
	if err != nil {
		return types.ApiBaseValueResp{}, err
	}
	if customer == nil {
		return types.ApiBaseValueResp{}, lib.NewUssAeErrorSimple(consts.ModuleCustomer, code.DataNotFound, fmt.Sprintf("customer not found for customerId:%d", customerID))
	}
	profile, err := s.customerProfileRepo.Find(ctx, req.CustomerId)
	if err != nil {
		return types.ApiBaseValueResp{}, err
	}
	if s.checkIDCard(profile, req) {
		return types.ApiBaseValueResp{}, lib.NewUssAeErrorSimple(consts.ModuleProfile, code.InvalidParam, "ID card is already verified and not allowed to change")
	}
	ci := model.CustomerImage{
		CustomerID:      req.CustomerId,
		IDCardFront:     sql.NullString{String: req.IdCardFront, Valid: true},
		IDCardBack:      sql.NullString{String: req.IdCardBack, Valid: true},
		IDSelfie:        sql.NullString{String: req.IdSelfie, Valid: true},
		ExtraImageOne:   sql.NullString{String: req.ExtraImageOne, Valid: true},
		ExtraImageTwo:   sql.NullString{String: req.ExtraImageTwo, Valid: true},
		ExtraImageThree: sql.NullString{String: req.ExtraImageThree, Valid: true},
	}
	if err := s.customerImageRepo.Upsert(ctx, ci); err != nil {
		return types.ApiBaseValueResp{}, err
	}

}

func (s *service) checkIDCard(profile model.CustomerProfile, req types.CustomerImageTO) bool {
	if profile.IDVerification != "Y" {
		return false
	}
	return s.checkImages(req)
}

func (s *service) checkImages(req types.CustomerImageTO) bool {
	images := []string{
		req.IdCardFront,
		req.IdCardBack,
		req.IdSelfie,
		req.ExtraImageOne,
		req.ExtraImageTwo,
		req.ExtraImageThree,
	}

	for _, img := range images {
		if img != "" {
			return true
		}
	}
	return false
}
