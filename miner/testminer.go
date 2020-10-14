package miner

import (
	"context"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/gen"
	"github.com/filecoin-project/go-address"
	lru "github.com/hashicorp/golang-lru"
)

func NewTestMiner(nextCh <-chan func(bool, error), addr address.Address) func(api.FullNode, gen.WinningPoStProver) *Miner {
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

func chanWaiter(next <-chan func(bool, error)) func(ctx context.Context, _ uint64) (func(bool, error), error) {
	return func(ctx context.Context, _ uint64) (func(bool, error), error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case cb := <-next:
			return cb, nil
		}
	}
}
