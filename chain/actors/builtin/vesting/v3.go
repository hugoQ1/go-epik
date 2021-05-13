package vesting

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/vesting"
	"github.com/ipfs/go-cid"
)

var _ State = (*state)(nil)

func load(store adt.Store, root cid.Cid) (State, error) {
	out := state{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state struct {
	vesting.State
	store adt.Store
}

func (s *state) Coinbase(coinbase address.Address, currEpoch abi.ChainEpoch) (*CoinbaseInfo, error) {
	vf, found, err := s.LoadVestingFunds(s.store, coinbase)
	if err != nil {
		return nil, err
	}
	if !found {
		return &CoinbaseInfo{
			Total:   big.Zero(),
			Vested:  big.Zero(),
			Vesting: big.Zero(),
		}, nil
	}

	vested := big.Add(vf.UnlockVestedFunds(currEpoch), vf.UnlockedBalance)
	vesting := big.Zero()
	for _, f := range vf.Funds {
		vesting = big.Add(vesting, f.Amount)
	}
	return &CoinbaseInfo{
		Total:   big.Add(vested, vesting),
		Vested:  vested,
		Vesting: vesting,
	}, nil
}

func (s *state) TotalLocked() abi.TokenAmount {
	return s.LockedFunds
}
