package vote

import (
	"fmt"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/vote"
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
	vote.State
	store adt.Store
}

func (s *state) Tally() (*Tally, error) {
	candidates, err := adt2.AsMap(s.store, s.Candidates)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]abi.TokenAmount)

	var amt abi.TokenAmount
	err = candidates.ForEach(&amt, func(k string) error {
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
	return &Tally{
		TotalVotes: s.State.TotalVotes,
		Candidates: ret,
	}, nil
}

func (s *state) VoterInfo(addr address.Address) (*VoterInfo, error) {
	voter, err := getVoter(s, addr)
	if err != nil {
		return nil, err
	}
	tally, err := adt2.AsMap(s.store, voter.Tally)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]abi.TokenAmount)

	var info vote.VotesInfo
	err = tally.ForEach(&info, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		ret[a.String()] = info.Votes.Copy()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &VoterInfo{
		Withdrawable: voter.Withdrawable,
		Candidates:   ret,
	}, nil
}

func getVoter(s *state, addr address.Address) (*vote.Voter, error) {
	if addr.Protocol() != address.ID {
		return nil, fmt.Errorf("not a ID-address")
	}

	voters, err := adt2.AsMap(s.store, s.Voters)
	if err != nil {
		return nil, err
	}

	var voter vote.Voter
	found, err := voters.Get(abi.AddrKey(addr), &voter)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("not found")
	}
	return &voter, nil
}
