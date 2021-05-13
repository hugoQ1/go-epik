package flowchmgr

import (
	"context"

	"github.com/filecoin-project/go-address"

	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/flowch"
)

type BestSpendableAPI interface {
	FlowchVoucherList(context.Context, address.Address) ([]*flowch.SignedVoucher, error)
	FlowchVoucherCheckSpendable(context.Context, address.Address, *flowch.SignedVoucher, []byte, []byte) (bool, error)
}

func BestSpendableByLane(ctx context.Context, api BestSpendableAPI, ch address.Address) (map[uint64]*flowch.SignedVoucher, error) {
	vouchers, err := api.FlowchVoucherList(ctx, ch)
	if err != nil {
		return nil, err
	}

	bestByLane := make(map[uint64]*flowch.SignedVoucher)
	for _, voucher := range vouchers {
		spendable, err := api.FlowchVoucherCheckSpendable(ctx, ch, voucher, nil, nil)
		if err != nil {
			return nil, err
		}
		if spendable {
			if bestByLane[voucher.Lane] == nil || voucher.Amount.GreaterThan(bestByLane[voucher.Lane].Amount) {
				bestByLane[voucher.Lane] = voucher
			}
		}
	}
	return bestByLane, nil
}
