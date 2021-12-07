package expertfund

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	ef3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
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
	ef3.State
	store adt.Store
}

func (s *state3) DataExpert(pieceCID cid.Cid) (address.Address, error) {
	pieceToExpert, _, err := s.State.GetPieceInfos(s.store, pieceCID)
	if err != nil {
		return address.Undef, err
	}

	return pieceToExpert[pieceCID], nil
}

func (s *state3) ListAllExperts() ([]address.Address, error) {
	expertMap, err := s.experts()
	if err != nil {
		return nil, err
	}

	var experts []address.Address
	err = expertMap.ForEach(nil, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		experts = append(experts, a)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return experts, nil
}

func (s *state3) experts() (adt.Map, error) {
	return adt3.AsMap(s.store, s.Experts, builtin3.DefaultHamtBitwidth)
}

func (s *state3) ExpertInfo(a address.Address) (*ExpertInfo, error) {
	return s.State.GetExpert(s.store, a)
}

func (s *state3) DisqualifiedExpertInfo(a address.Address) (*DisqualifiedExpertInfo, error) {
	info, _, err := s.State.GetDisqualifiedExpertInfo(s.store, a)
	return info, err
}

func (s *state3) Reward(epoch abi.ChainEpoch, a address.Address) (*ExpertReward, error) {
	reward, err := s.State.Reward(s.store, epoch, a)
	if err != nil {
		return nil, err
	}
	vestingFund, err := adt3.AsMap(s.store, reward.VestingFunds, builtin3.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	var amount abi.TokenAmount
	vestingFundsMap := make(map[abi.ChainEpoch]abi.TokenAmount)
	err = vestingFund.ForEach(&amount, func(k string) error {
		epoch, err := abi.ParseIntKey(k)
		if err != nil {
			return xerrors.Errorf("failed to parse vestingFund key: %w", err)
		}
		vestingFundsMap[abi.ChainEpoch(epoch)] = amount
		return nil
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to iterate vestingFund: %w", err)
	}
	return &ExpertReward{
		RewardDebt:    reward.RewardDebt,
		LockedFunds:   reward.LockedFunds,
		UnlockedFunds: reward.UnlockedFunds,
		VestingFunds:  vestingFundsMap,
	}, nil
}

func (s *state3) DataThreshold() uint64 {
	return s.DataStoreThreshold
}

func (s *state3) DailyThreshold() uint64 {
	return s.DailyImportSizeThreshold
}
