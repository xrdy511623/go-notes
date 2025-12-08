package customerimage

import (
	"context"
	"fmt"

	"tcg-uss-ae-go-sp/internal/apiserver/types"
	"tcg-uss-ae-go-sp/internal/pkg/code"
	"tcg-uss-ae-go-sp/internal/pkg/consts"
	"tcg-uss-ae-go-sp/internal/pkg/lib"
	"tcg-uss-ae-go-sp/utils"
)

func (s *service) GetCustomerImageByID(ctx context.Context, customerID int64) (types.ApiBaseValueResp, error) {
	customer, err := s.customerImmutableCache.GetById(ctx, customerID)
	if err != nil {
		return types.ApiBaseValueResp{}, err
	}
	if customer == nil {
		return types.ApiBaseValueResp{}, lib.NewUssAeErrorSimple(consts.ModuleCustomer, code.DataNotFound, fmt.Sprintf("customer not found for customerId:%d", customerID))
	}
	customerImage, err := s.customerImageRepo.Find(ctx, customerID)
	if err != nil {
		return types.ApiBaseValueResp{}, err
	}
	if customerImage.CustomerID == 0 {
		customerImage.CustomerID = customerID
	}
	res := types.CustomerImage{
		CustomerID:      customerID,
		IDCardFront:     utils.StructValueFromNullString(customerImage.IDCardFront),
		IDCardBack:      utils.StructValueFromNullString(customerImage.IDCardBack),
		Avatar:          utils.StructValueFromNullString(customerImage.Avatar),
		IDSelfie:        utils.StructValueFromNullString(customerImage.IDSelfie),
		ExtraImageOne:   utils.StructValueFromNullString(customerImage.ExtraImageOne),
		ExtraImageTwo:   utils.StructValueFromNullString(customerImage.ExtraImageTwo),
		ExtraImageThree: utils.StructValueFromNullString(customerImage.ExtraImageThree),
	}

	return types.ApiBaseValueResp{
		Success: true,
		Value:   res,
	}, nil
}
