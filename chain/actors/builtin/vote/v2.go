package vote

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
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
	candidates, err := adt2.AsMap(s.store, s.Candidates, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	votes := make(map[string]abi.TokenAmount)
	blocked := make(map[string]bool)
	var out vote.Candidate
	err = candidates.ForEach(&out, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		votes[a.String()] = out.Votes
		blocked[a.String()] = out.IsBlocked()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &Tally{
		TotalVotes:       s.State.CurrEpochEffectiveVotes,
		FallbackReceiver: s.State.FallbackReceiver,
		Candidates:       votes,
		Blocked:          blocked,
	}, nil
}

func (s *state) VoterInfo(vaddr address.Address, currEpoch abi.ChainEpoch, currBalance abi.TokenAmount) (*VoterInfo, error) {
	voter, err := s.EstimateSettle(s.store, vaddr, currEpoch, currBalance)
	if err != nil {
		return nil, err
	}

	tally, err := adt2.AsMap(s.store, voter.Tally, builtin.DefaultHamtBitwidth)
	if err != nil {
		return nil, err
	}

	// Valid votes of each candidate that vaddr voted
	cands := make(map[string]abi.TokenAmount)
	unlocking := big.Zero()
	unlocked := big.Zero()
	var info vote.VotesInfo
	err = tally.ForEach(&info, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		cands[a.String()] = info.Votes
		if !info.RescindingVotes.IsZero() {
			if currEpoch <= info.LastRescindEpoch+vote.RescindingUnlockDelay {
				unlocking = big.Add(unlocking, info.RescindingVotes)
			} else {
				unlocked = big.Add(unlocked, info.RescindingVotes)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &VoterInfo{
		UnlockingVotes:      unlocking,
		UnlockedVotes:       unlocked,
		WithdrawableRewards: voter.Withdrawable,
		Candidates:          cands,
	}, nil
}
