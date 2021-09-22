package vote

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func init() {
	builtin.RegisterActorState(builtin2.VoteFundActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load(store, root)
	})
}

var (
	Address = builtin2.VoteFundActorAddr
	Methods = builtin2.MethodsVote
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.VoteFundActorCodeID:
		return load(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	Tally() (*Tally, error)
	VoterInfo(addr address.Address, currEpoch abi.ChainEpoch, actorBalance abi.TokenAmount) (*VoterInfo, error)
	ListVoterInfos(currEpoch abi.ChainEpoch, actorBalance abi.TokenAmount) ([]*VoterInfo, error)
}

type Tally struct {
	TotalVotes       abi.TokenAmount
	UnownedFunds     abi.TokenAmount
	FallbackReceiver address.Address
	Candidates       map[string]abi.TokenAmount // key is candidate address
	Blocked          map[string]bool
}

type VoterInfo struct {
	Voter               address.Address
	UnlockingVotes      abi.TokenAmount
	UnlockedVotes       abi.TokenAmount
	WithdrawableRewards abi.TokenAmount
	Candidates          map[string]abi.TokenAmount // key is candidate address
}
