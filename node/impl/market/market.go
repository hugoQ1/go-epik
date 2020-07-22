package market

import (
	"context"

	"go.uber.org/fx"

	"github.com/EpiK-Protocol/go-epik/chain/market"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
)

type MarketAPI struct {
	fx.In

	FMgr *market.FundMgr
}

func (a *MarketAPI) MarketEnsureAvailable(ctx context.Context, addr, wallet address.Address, amt types.BigInt) (cid.Cid, error) {
	return a.FMgr.EnsureAvailable(ctx, addr, wallet, amt)
}
