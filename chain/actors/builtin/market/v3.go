package market

import (
	"bytes"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	market3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
	adt3 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
)

var _ State = (*state3)(nil)

func load3(store adt.Store, root cid.Cid) (State, error) {
	out := state3{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state3 struct {
	market3.State
	store adt.Store
}

func (s *state3) TotalLocked() (abi.TokenAmount, error) {
	// fml := types.BigAdd(s.TotalClientLockedCollateral, s.TotalProviderLockedCollateral)
	// fml = types.BigAdd(fml, s.TotalClientStorageFee)
	// return fml, nil
	return big.Zero(), nil
}

func (s *state3) BalancesChanged(otherState State) (bool, error) {
	otherState2, ok := otherState.(*state3)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.EscrowTable.Equals(otherState2.State.EscrowTable) || !s.State.LockedTable.Equals(otherState2.State.LockedTable), nil
}

func (s *state3) StatesChanged(otherState State) (bool, error) {
	otherState2, ok := otherState.(*state3)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.States.Equals(otherState2.State.States), nil
}

func (s *state3) States() (DealStates, error) {
	stateArray, err := adt3.AsArray(s.store, s.State.States, market3.StatesAmtBitwidth)
	if err != nil {
		return nil, err
	}
	return &dealStates3{stateArray}, nil
}

func (s *state3) ProposalsChanged(otherState State) (bool, error) {
	otherState2, ok := otherState.(*state3)
	if !ok {
		// there's no way to compare different versions of the state, so let's
		// just say that means the state of balances has changed
		return true, nil
	}
	return !s.State.Proposals.Equals(otherState2.State.Proposals), nil
}

func (s *state3) Proposals() (DealProposals, error) {
	proposalArray, err := adt3.AsArray(s.store, s.State.Proposals, market3.ProposalsAmtBitwidth)
	if err != nil {
		return nil, err
	}
	return &dealProposals3{proposalArray}, nil
}

func (s *state3) EscrowTable() (BalanceTable, error) {
	bt, err := adt3.AsBalanceTable(s.store, s.State.EscrowTable)
	if err != nil {
		return nil, err
	}
	return &balanceTable3{bt}, nil
}

func (s *state3) LockedTable() (BalanceTable, error) {
	bt, err := adt3.AsBalanceTable(s.store, s.State.LockedTable)
	if err != nil {
		return nil, err
	}
	return &balanceTable3{bt}, nil
}

func (s *state3) Quotas() (Quotas, error) {
	quotas, err := adt3.AsMap(s.store, s.State.Quotas, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}
	return &quotasAccesor{
		quotas,
		s.State.InitialQuota,
	}, nil
}

func (s *state3) DataIndexes() (DataIndexes, error) {
	indexes, err := market3.AsIndexMultimap(s.store, s.State.DataIndexesByEpoch, builtin.DefaultHamtBitwidth, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}
	return indexes, nil
}

func (s *state3) HasPendingPiece(pieceCids []cid.Cid) (bool, error) {
	pendings, err := adt3.AsMap(s.store, s.State.PendingProposals, builtin.DefaultHamtBitwidth)
	if err != nil {
		return false, err
	}

	all := make(map[cid.Cid]struct{})
	var dataIndex market3.ProposalDataIndex
	err = pendings.ForEach(&dataIndex, func(k string) error {
		all[dataIndex.Index.PieceCID] = struct{}{}
		return nil
	})
	if err != nil {
		return false, err
	}

	for _, pieceCid := range pieceCids {
		if _, ok := all[pieceCid]; ok {
			return true, nil
		}
	}
	return false, nil
}

type balanceTable3 struct {
	*adt3.BalanceTable
}

func (bt *balanceTable3) ForEach(cb func(address.Address, abi.TokenAmount) error) error {
	asMap := (*adt3.Map)(bt.BalanceTable)
	var ta abi.TokenAmount
	return asMap.ForEach(&ta, func(key string) error {
		a, err := address.NewFromBytes([]byte(key))
		if err != nil {
			return err
		}
		return cb(a, ta)
	})
}

type dealStates3 struct {
	adt.Array
}

func (s *dealStates3) Get(dealID abi.DealID) (*DealState, bool, error) {
	var out market3.DealState
	found, err := s.Array.Get(uint64(dealID), &out)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	ds := (DealState)(out)
	return &ds, true, nil
}

func (s *dealStates3) ForEach(cb func(dealID abi.DealID, ds DealState) error) error {
	var ds market3.DealState
	return s.Array.ForEach(&ds, func(idx int64) error {
		return cb(abi.DealID(idx), (DealState)(ds))
	})
}

func (s *dealStates3) decode(val *cbg.Deferred) (*DealState, error) {
	var ds market3.DealState
	if err := ds.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return nil, err
	}
	out := (DealState)(ds)
	return &out, nil
}

func (s *dealStates3) array() adt.Array {
	return s.Array
}

type dealProposals3 struct {
	adt.Array
}

func (s *dealProposals3) Get(dealID abi.DealID) (*DealProposal, bool, error) {
	var out market3.DealProposal
	found, err := s.Array.Get(uint64(dealID), &out)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	proposal := (DealProposal)(out)
	return &proposal, true, nil
}

func (s *dealProposals3) ForEach(cb func(dealID abi.DealID, dp DealProposal) error) error {
	var out market3.DealProposal
	return s.Array.ForEach(&out, func(idx int64) error {
		return cb(abi.DealID(idx), (DealProposal)(out))
	})
}

func (s *dealProposals3) decode(val *cbg.Deferred) (*DealProposal, error) {
	var out market3.DealProposal
	if err := out.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return nil, err
	}
	dp := (DealProposal)(out)
	return &dp, nil
}

func (s *dealProposals3) array() adt.Array {
	return s.Array
}

type quotasAccesor struct {
	adt.Map
	initial int64
}

func (a *quotasAccesor) InitialQuota() int64 {
	return a.initial
}

func (a *quotasAccesor) RemainingQuota(pieceCID cid.Cid) (int64, error) {
	var out cbg.CborInt
	found, err := a.Map.Get(abi.CidKey(pieceCID), &out)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, fmt.Errorf("piece not found: %s", pieceCID)
	}
	return int64(out), nil
}
