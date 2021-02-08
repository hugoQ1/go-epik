package knowledge

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/knowledge"
	"github.com/ipfs/go-cid"

	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
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
	knowledge.State
	store adt.Store
}

func (s *state) Info() (*Info, error) {
	tally, err := adt2.AsMap(s.store, s.Tally)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]abi.TokenAmount)

	var amt abi.TokenAmount
	err = tally.ForEach(&amt, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		ret[a.String()] = amt.Copy()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Info{
		Payee: s.State.Payee,
		Tally: ret,
	}, nil
}
