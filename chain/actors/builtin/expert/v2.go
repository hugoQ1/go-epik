package expert

import (
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"

	expert2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/expert"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
)

var _ State = (*state2)(nil)

func load2(store adt.Store, root cid.Cid) (State, error) {
	out := state2{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state2 struct {
	expert2.State
	store adt.Store
}

func (s *state2) Info() (*ExpertInfo, error) {
	return s.State.GetInfo(s.store)
}

func (s *state2) Datas() ([]*DataOnChainInfo, error) {
	var datas []*DataOnChainInfo
	ds, err := adt2.AsMap(s.store, s.State.Datas)
	if err != nil {
		return nil, err
	}

	var info DataOnChainInfo
	err = ds.ForEach(&info, func(k string) error {
		cp := info
		datas = append(datas, &cp)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return datas, nil
}

func (s *state2) Data(pieceCID cid.Cid) (*DataOnChainInfo, error) {
	datas, err := adt2.AsMap(s.store, s.State.Datas)
	if err != nil {
		return nil, err
	}

	var info DataOnChainInfo
	found, err := datas.Get(adt2.StringKey(pieceCID.String()), &info)
	if err != nil {
		return nil, xerrors.Errorf("failed to get expert data %s: %w", pieceCID, err)
	}
	if !found {
		return nil, nil
	}
	return &info, nil
}
