package retrieval

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	retrieval2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/retrieval"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

var _ State = (*state)(nil)

func load2(store adt.Store, root cid.Cid) (State, error) {
	out := state{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state struct {
	retrieval2.State
	store adt.Store
}

func (s *state) PledgesInfo(addr address.Address) (map[address.Address]abi.TokenAmount, error) {
	pledgesMap, err := adt.AsMap(s.store, s.Pledges, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("failed to load pledges:%v", err)
	}

	var pledge PledgeState
	found, err := pledgesMap.Get(abi.AddrKey(addr), &pledge)
	if err != nil {
		return nil, xerrors.Errorf("failed to get pledger info:%v", err)
	}
	if !found {
		return nil, xerrors.Errorf("failed to find the pledge:%s", addr)
	}
	tmap, err := adt.AsMap(s.store, pledge.Targets, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("failed to load pledge target:%v", err)
	}

	targets := make(map[address.Address]abi.TokenAmount)
	var outAmount abi.TokenAmount
	err = tmap.ForEach(&outAmount, func(key string) error {
		addr, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		targets[addr] = outAmount
		return nil
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to load pledge amount:%v", err)
	}
	return targets, nil
}

func (s *state) StateInfo(fromAddr address.Address) (*RetrievalState, error) {
	info, err := s.State.StateInfo(s.store, fromAddr)
	if err != nil {
		return nil, err
	}
	mmap, err := adt.AsMap(s.store, info.Miners, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}
	miners, err := mmap.CollectKeys()
	if err != nil {
		return nil, err
	}
	var addrs []address.Address
	for _, miner := range miners {
		addr, err := address.NewFromBytes([]byte(miner))
		if err != nil {
			return nil, err
		}
		addrs = append(addrs, addr)
	}
	return &RetrievalState{
		BindMiners: addrs,
		Amount:     info.Amount,
		EpochDate:  info.EpochDate,
		DateSize:   info.DailyDataSize,
	}, nil
}

func (s *state) DayExpend(epoch abi.ChainEpoch, fromAddr address.Address) (abi.TokenAmount, error) {
	return s.State.DayExpend(s.store, epoch, fromAddr)
}

func (s *state) LockedState(fromAddr address.Address, out *LockedState) (bool, error) {
	lockedMap, err := adt.AsMap(s.store, s.State.LockedTable, builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, err
	}
	found, err := lockedMap.Get(abi.AddrKey(fromAddr), out)
	if err != nil {
		return false, err
	}
	return found, nil
}

func (s *state) TotalCollateral() (abi.TokenAmount, error) {
	return s.State.TotalCollateral, nil
}

func (s *state) TotalRetrievalReward() (abi.TokenAmount, error) {
	return s.State.TotalRetrievalReward, nil
}

func (s *state) PendingReward() (abi.TokenAmount, error) {
	return s.State.PendingReward, nil
}

func (s *state) ForEachState(cb func(addr address.Address, state *RetrievalState) error) error {
	stateMap, err := adt.AsMap(s.store, s.RetrievalStates, builtin.DefaultHamtBitwidth)
	if err != nil {
		return err
	}
	addrStrs, err := stateMap.CollectKeys()
	if err != nil {
		return err
	}
	for _, str := range addrStrs {
		addr, err := address.NewFromBytes([]byte(str))
		if err != nil {
			return err
		}
		state, err := s.StateInfo(addr)
		if err != nil {
			return err
		}
		return cb(addr, state)
	}
	return nil
}
