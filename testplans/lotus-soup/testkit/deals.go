package testkit

import (
	"context"
	"fmt"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"

	tstats "github.com/EpiK-Protocol/go-epik/tools/stats"
)

func StartDeal(t *TestEnvironment, ctx context.Context, maMsg MinerAddressesMsg, client api.FullNode, fcid cid.Cid, fastRetrieval bool) *cid.Cid {
	addr, err := client.WalletDefaultAddress(ctx)
	if err != nil {
		panic(err)
	}

	experts, err := client.StateListExperts(ctx, types.EmptyTSK)
	if err != nil {
		panic(err)
	}
	if len(experts) == 0 {
		panic("no experts")
	}
	t.RecordMessage("expert list: %v", experts)

	expertInfo, err := client.StateExpertInfo(ctx, experts[0], types.EmptyTSK)
	if err != nil {
		panic(err)
	}
	t.RecordMessage("start deal with expert %s ( owner %s ), wallet %s, fcid %s", experts[0], expertInfo.Owner, addr, fcid.String())

	deal, err := client.ClientStartDeal(ctx, &api.StartDealParams{
		Data: &storagemarket.DataRef{
			TransferType: storagemarket.TTGraphsync,
			Root:         fcid,
			Expert:       experts[0].String(),
		},
		Wallet: addr,
		Miner:  maMsg.MinerActorAddr,
		// EpochPrice:        types.NewInt(1000),
		// MinBlocksDuration: 640000,
		DealStartEpoch: 200,
		FastRetrieval:  fastRetrieval,
	})
	if err != nil {
		panic(err)
	}
	return deal
}

func WaitDealSealed(t *TestEnvironment, ctx context.Context, client api.FullNode, deal *cid.Cid, fcid cid.Cid) {
	height := 0
	headlag := 3

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tipsetsCh, err := tstats.GetTips(cctx, client, abi.ChainEpoch(height), headlag)
	if err != nil {
		panic(err)
	}

	experts, err := client.StateListExperts(ctx, types.EmptyTSK)
	if err != nil {
		panic(err)
	}

	imported := false
	for tipset := range tipsetsCh {
		if !imported {
			infos, err := client.StateExpertDatas(ctx, experts[0], nil, false, types.EmptyTSK)
			if err != nil {
				panic(err)
			}

			for _, info := range infos {
				if info.PieceID == fcid.String() {
					imported = true
					t.RecordMessage("piece %s of expert %s imported", info.PieceID, experts[0])
					break
				}
			}
		}

		if !imported {
			continue
		}

		di, err := client.ClientGetDealInfo(ctx, *deal)
		if err != nil {
			panic(err)
		}
		switch di.State {
		case storagemarket.StorageDealProposalRejected:
			panic("deal rejected")
		case storagemarket.StorageDealFailing:
			panic("deal failed")
		case storagemarket.StorageDealError:
			panic(fmt.Sprintf("deal errored %s", di.Message))
		case storagemarket.StorageDealActive:
			t.RecordMessage("height %d, completed deal: %s", tipset.Height(), di)
			return
		}

		t.RecordMessage("height %d, deal state: %s", tipset.Height(), storagemarket.DealStates[di.State])
	}
}
