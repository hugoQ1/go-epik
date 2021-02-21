package reward

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	reward3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
)

func init() {
	// builtin.RegisterActorState(builtin2.RewardActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
	// 	return load2(store, root)
	// })
	builtin.RegisterActorState(builtin3.RewardActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load3(store, root)
	})
}

var (
	Address = builtin3.RewardActorAddr
	Methods = builtin3.MethodsReward
)

type AwardBlockRewardReturn = reward3.AwardBlockRewardReturn
type AwardBlockRewardParams = reward3.AwardBlockRewardParams

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	// case builtin2.RewardActorCodeID:
	// 	return load2(store, act.Head)
	// }
	case builtin3.RewardActorCodeID:
		return load3(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	ThisEpochReward() (abi.TokenAmount, error)
	TotalMined() (abi.TokenAmount, error)
	TotalMinedDetail() (*TotalMinedDetail, error)
}

type TotalMinedDetail struct {
	TotalExpertReward       abi.TokenAmount // to expert fund actor
	TotalVoteReward         abi.TokenAmount // to vote fund actor
	TotalKnowledgeReward    abi.TokenAmount // to knowledge fund actor
	TotalRetrievalReward    abi.TokenAmount // to retrieval fund actor
	TotalStoragePowerReward abi.TokenAmount // to block miners
}
