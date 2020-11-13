package impl

import (
	"context"

	logging "github.com/ipfs/go-log/v2"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/node/impl/client"
	"github.com/EpiK-Protocol/go-epik/node/impl/common"
	"github.com/EpiK-Protocol/go-epik/node/impl/full"
	"github.com/EpiK-Protocol/go-epik/node/impl/market"
	"github.com/EpiK-Protocol/go-epik/node/impl/paych"
	"github.com/EpiK-Protocol/go-epik/node/modules/dtypes"
)

var log = logging.Logger("node")

type FullNodeAPI struct {
	common.CommonAPI
	full.ChainAPI
	client.API
	full.MpoolAPI
	full.GasAPI
	market.MarketAPI
	paych.PaychAPI
	full.StateAPI
	full.MsigAPI
	full.WalletAPI
	full.SyncAPI
	full.BeaconAPI

	DS dtypes.MetadataDS
}

func (n *FullNodeAPI) CreateBackup(ctx context.Context, fpath string) error {
	return backup(n.DS, fpath)
}

var _ api.FullNode = &FullNodeAPI{}
