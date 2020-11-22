package retrieval

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/filecoin-project/go-state-types/abi"
	retrieval2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/retrieval"
	"github.com/ipfs/go-cid"
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

func (s *state) TotalCollateral() (abi.TokenAmount, error) {
	return s.State.TotalCollateral, nil
}
