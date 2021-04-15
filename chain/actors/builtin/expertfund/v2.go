package expertfund

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
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
	infos, err := s.State.GetDataInfos(s.store, pieceCID)
	if err != nil {
		return address.Undef, err
	}

	return infos[0].Expert, nil
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

func (s *state3) ExpertFundInfo(a address.Address) (*ExpertFundInfo, error) {
	return s.State.GetExpert(s.store, a)
}

func (s *state3) Threshold() uint64 {
	return s.DataStoreThreshold
}
