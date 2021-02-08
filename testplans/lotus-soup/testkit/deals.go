package testkit

import (
	"context"
	"fmt"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	tstats "github.com/EpiK-Protocol/go-epik/tools/stats"
)

func StartDeal(t *TestEnvironment, ctx context.Context, maMsg MinerAddressesMsg, client api.FullNode, fcid cid.Cid, fastRetrieval bool) *cid.Cid {
	addr, err := client.WalletDefaultAddress(ctx)
	if err != nil {
		panic(err)
	}

	expert := getExpertByOwner(t, ctx, client, addr)

	ds, err := client.ClientDealPieceCID(ctx, fcid)
	if err != nil {
		panic(err)
	}

	// register file
	failCount := 0
	for {
		mcid, err := client.ClientExpertRegisterFile(ctx, &api.ExpertRegisterFileParams{
			Expert:    expert,
			PieceID:   ds.PieceCID,
			PieceSize: ds.PieceSize,
		})
		if err != nil {
			panic(err)
		}
		t.RecordMessage("send register file message: %s", mcid)

		mlk, err := client.StateWaitMsg(ctx, *mcid, 2)
		if err != nil {
			panic(err)
		}
		if mlk.Receipt.ExitCode == 0 {
			t.RecordMessage("register file success: %s", mcid)

			infos, err := client.StateExpertDatas(ctx, expert, nil, false, types.EmptyTSK)
			if err != nil {
				panic(err)
			}

			if len(infos) == 0 {
				panic(xerrors.Errorf("registered but no data: %s", expert))
			}

			break
		}
		t.RecordFailure(fmt.Errorf("message %s return failed code: %d", mcid, mlk.Receipt.ExitCode))
		failCount++
		if failCount > 10 {
			panic("failed to register file: " + fcid.String())
		}
	}

	t.RecordMessage("start deal with wallet %s, fcid %s, piece %s, piece size %d", addr, fcid.String(), ds.PieceCID, ds.PayloadSize)

	deal, err := client.ClientStartDeal(ctx, &api.StartDealParams{
		Data: &storagemarket.DataRef{
			TransferType: storagemarket.TTGraphsync,
			Root:         fcid,
			Expert:       addr.String(),
		},
		Wallet:         addr,
		Miner:          maMsg.MinerActorAddr,
		DealStartEpoch: 200,
		FastRetrieval:  fastRetrieval,
	})
	if err != nil {
		panic(err)
	}
	return deal
}

func WaitDealSealed(t *TestEnvironment, ctx context.Context, client api.FullNode, deal *cid.Cid) {
	height := 0
	headlag := 3

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tipsetsCh, err := tstats.GetTips(cctx, client, abi.ChainEpoch(height), headlag)
	if err != nil {
		panic(err)
	}

	for tipset := range tipsetsCh {
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

func getExpertByOwner(t *TestEnvironment, ctx context.Context, client api.FullNode, owner address.Address) address.Address {
	ida, err := client.StateLookupID(ctx, owner, types.EmptyTSK)
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

	var myExpert address.Address
	for _, e := range experts {
		expertInfo, err := client.StateExpertInfo(ctx, e, types.EmptyTSK)
		if err != nil {
			panic(err)
		}
		if expertInfo.Owner == ida {
			myExpert = e
			break
		}
		t.RecordMessage("expert %s owner %s", e, expertInfo.Owner)
	}
	if myExpert == address.Undef {
		panic("default wallet not an expert owner: " + ida.String())
	}
	return myExpert
}
