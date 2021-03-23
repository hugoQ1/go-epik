package reward

/* import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
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
	reward.State
	store adt.Store
}

func (s *state2) ThisEpochReward() (abi.TokenAmount, error) {
	return s.State.ThisEpochReward, nil
}

func (s *state2) TotalMined() (abi.TokenAmount, error) {
	return big.Sum(
		s.State.TotalExpertReward,
		s.State.TotalKnowledgeReward,
		s.State.TotalRetrievalReward,
		s.State.TotalStoragePowerReward,
		s.State.TotalVoteReward,
	), nil
}

func (s *state2) TotalMinedDetail() (*TotalMinedDetail, error) {
	return &TotalMinedDetail{
		TotalExpertReward:       s.State.TotalExpertReward,
		TotalKnowledgeReward:    s.State.TotalKnowledgeReward,
		TotalRetrievalReward:    s.State.TotalRetrievalReward,
		TotalStoragePowerReward: s.State.TotalStoragePowerReward,
		TotalVoteReward:         s.State.TotalVoteReward,
	}, nil
} */
