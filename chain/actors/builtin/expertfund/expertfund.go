package expertfund

import (
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	expertfund2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
)

func init() {
	builtin.RegisterActorState(builtin3.ExpertFundActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load3(store, root)
	})
}

var (
	Address = builtin3.ExpertFundActorAddr
	Methods = builtin3.MethodsExpertFunds
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin3.ExpertFundActorCodeID:
		return load3(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	DataExpert(cid.Cid) (address.Address, error)
	ListAllExperts() ([]address.Address, error)
	ExpertInfo(address.Address) (*ExpertInfo, error)
	DisqualifiedExpertInfo(address.Address) (*DisqualifiedExpertInfo, error)
	Reward(abi.ChainEpoch, address.Address) (*ExpertReward, error)
	DataThreshold() uint64
	DailyThreshold() uint64
}

type ExpertInfo = expertfund2.ExpertInfo
type DisqualifiedExpertInfo = expertfund2.DisqualifiedExpertInfo

// type ExpertReward = expertfund2.ExpertReward
type ExpertReward struct {
	RewardDebt abi.TokenAmount

	LockedFunds abi.TokenAmount // Total rewards and added funds locked in vesting table

	UnlockedFunds abi.TokenAmount

	VestingFunds map[abi.ChainEpoch]abi.TokenAmount
}
