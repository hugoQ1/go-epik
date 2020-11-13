package miner

import (
	"context"

	lru "github.com/hashicorp/golang-lru"
	ds "github.com/ipfs/go-datastore"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/gen"
	"github.com/EpiK-Protocol/go-epik/chain/gen/slashfilter"
	"github.com/EpiK-Protocol/go-epik/journal"
)

type MineReq struct {
	InjectNulls abi.ChainEpoch
	Done        func(bool, abi.ChainEpoch, error)
}

func NewTestMiner(nextCh <-chan MineReq, addr address.Address) func(api.FullNode, gen.WinningPoStProver) *Miner {
	return func(api api.FullNode, epp gen.WinningPoStProver) *Miner {
		arc, err := lru.NewARC(10000)
		if err != nil {
			panic(err)
		}

		data, err := lru.NewARC(1000)
		if err != nil {
			panic(err)
		}
		retrievals, err := lru.NewARC(1000)
		if err != nil {
			panic(err)
		}
		deals, err := lru.NewARC(1000)
		if err != nil {
			panic(err)
		}

		m := &Miner{
			api:               api,
			waitFunc:          chanWaiter(nextCh),
			epp:               epp,
			minedBlockHeights: arc,
			address:           addr,
			sf:                slashfilter.New(ds.NewMapDatastore()),
			journal:           journal.NilJournal(),
			minerData: &MinerData{
				api:        api,
				address:    addr,
				dataRefs:   data,
				retrievals: retrievals,
				deals:      deals,
			},
		}

		if err := m.Start(context.TODO()); err != nil {
			panic(err)
		}
		return m
	}
}

func chanWaiter(next <-chan MineReq) func(ctx context.Context, _ uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error) {
	return func(ctx context.Context, _ uint64) (func(bool, abi.ChainEpoch, error), abi.ChainEpoch, error) {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		case req := <-next:
			return req.Done, req.InjectNulls, nil
		}
	}
}
