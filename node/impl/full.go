package impl

import (
	logging "github.com/ipfs/go-log/v2"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/node/impl/client"
	"github.com/EpiK-Protocol/go-epik/node/impl/common"
	"github.com/EpiK-Protocol/go-epik/node/impl/full"
	"github.com/EpiK-Protocol/go-epik/node/impl/market"
	"github.com/EpiK-Protocol/go-epik/node/impl/paych"
)

var log = logging.Logger("node")

type FullNodeAPI struct {
	common.CommonAPI
	full.ChainAPI
	client.API
	full.MpoolAPI
	market.MarketAPI
	paych.PaychAPI
	full.StateAPI
	full.MsigAPI
	full.WalletAPI
	full.SyncAPI
}

var _ api.FullNode = &FullNodeAPI{}
