package govern

import (
	"fmt"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/govern"
	"github.com/ipfs/go-cid"

	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
)

func load(store adt.Store, root cid.Cid) (State, error) {
	out := state{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state struct {
	govern.State
	store adt.Store
}

func (s *state) Supervior() address.Address {
	return s.Supervisor
}

func (s *state) Governor(addr address.Address) (*GovernorInfo, error) {
	if addr.Protocol() != address.ID {
		return nil, fmt.Errorf("not a ID-address")
	}

	governors, err := adt2.AsMap(s.store, s.Governors, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	var ga govern.GrantedAuthorities
	found, err := governors.Get(abi.AddrKey(addr), &ga)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("not found")
	}

	authorities, err := convert(s.store, ga)
	if err != nil {
		return nil, err
	}

	return &GovernorInfo{
		Address:     addr,
		Authorities: authorities,
	}, nil
}

func (s *state) ListGovrnors() ([]*GovernorInfo, error) {
	governors, err := adt2.AsMap(s.store, s.Governors, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	var ret []*GovernorInfo

	var ga govern.GrantedAuthorities
	err = governors.ForEach(&ga, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}

		authorities, err := convert(s.store, ga)
		if err != nil {
			return err
		}
		ret = append(ret, &GovernorInfo{
			Address:     a,
			Authorities: authorities,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func convert(store adt.Store, ga govern.GrantedAuthorities) ([]Authority, error) {
	codeMethods, err := adt2.AsMap(store, ga.CodeMethods, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	ret := make([]Authority, 0)

	var bf bitfield.BitField
	err = codeMethods.ForEach(&bf, func(k string) error {
		_, codeId, err := cid.CidFromBytes([]byte(k))
		if err != nil {
			return err
		}

		auth := Authority{ActorCodeID: codeId}
		defer func() {
			ret = append(ret, auth)
		}()

		return bf.ForEach(func(i uint64) error {
			auth.Methods = append(auth.Methods, abi.MethodNum(i))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}
