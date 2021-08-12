package govern

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	power3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func init() {
	builtin.RegisterActorState(builtin2.GovernActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load(store, root)
	})
}

var (
	Address = builtin2.GovernActorAddr
	Methods = builtin2.MethodsGovern
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.GovernActorCodeID:
		return load(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	Supervior() address.Address
	Governor(address.Address) (*GovernorInfo, error)
	ListGovrnors() ([]*GovernorInfo, error)
}

type GovernorInfo struct {
	Address     address.Address
	Authorities []Authority
}

type Authority struct {
	ActorCodeID cid.Cid
	Methods     []abi.MethodNum
}

type GovParams struct {
	MinersPoStRatio     power3.WdPoStRatio
	MinersPledgePeriod  abi.ChainEpoch
	MarketInitialQuota  int64
	ExpertfundThreshold uint64
	KnowledgePayee      address.Address
}
