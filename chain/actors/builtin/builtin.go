package builtin

import (
	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	smoothing2 "github.com/filecoin-project/specs-actors/v2/actors/util/smoothing"
)

var SystemActorAddr = builtin.SystemActorAddr
var BurntFundsActorAddr = builtin.BurntFundsActorAddr
var CronActorAddr = builtin.CronActorAddr

// var SaftAddress = makeAddress("t0122")
// var ReserveAddress = makeAddress("t090")
// var RootVerifierAddress = makeAddress("t080")
var GovernAddress = makeAddress("t090")      // TODO: defined in actors
var TeamAddress = makeAddress("t091")        // TODO: defined in actors
var FoundationAddress = makeAddress("t092")  // TODO: defined in actors
var FundraisingAddress = makeAddress("t093") // TODO: defined in actors

var (
	ExpectedLeadersPerEpoch = builtin.ExpectedLeadersPerEpoch
)

const (
	EpochDurationSeconds = builtin.EpochDurationSeconds
	EpochsInDay          = builtin.EpochsInDay
	SecondsInDay         = builtin.SecondsInDay
)

const (
	MethodSend        = builtin.MethodSend
	MethodConstructor = builtin.MethodConstructor
)

// TODO: Why does actors have 2 different versions of this?
type SectorInfo = proof.SectorInfo
type PoStProof = proof.PoStProof
type FilterEstimate = smoothing2.FilterEstimate

/* func FromV0FilterEstimate(v0 smoothing2.FilterEstimate) FilterEstimate {
	return (FilterEstimate)(v0)
}

// Doesn't change between actors v0 and v1
func QAPowerForWeight(size abi.SectorSize, duration abi.ChainEpoch, dealWeight, verifiedWeight abi.DealWeight) abi.StoragePower {
	return miner2.QAPowerForWeight(size, duration, dealWeight, verifiedWeight)
}

func FromV2FilterEstimate(v1 smoothing2.FilterEstimate) FilterEstimate {
	return (FilterEstimate)(v1)
} */

type ActorStateLoader func(store adt.Store, root cid.Cid) (cbor.Marshaler, error)

var ActorStateLoaders = make(map[cid.Cid]ActorStateLoader)

func RegisterActorState(code cid.Cid, loader ActorStateLoader) {
	ActorStateLoaders[code] = loader
}

func Load(store adt.Store, act *types.Actor) (cbor.Marshaler, error) {
	loader, found := ActorStateLoaders[act.Code]
	if !found {
		return nil, xerrors.Errorf("unknown actor code %s", act.Code)
	}
	return loader(store, act.Head)
}

func ActorNameByCode(c cid.Cid) string {
	switch {
	case builtin.IsBuiltinActor(c):
		return builtin.ActorNameByCode(c)
	default:
		return "<unknown>"
	}
}

func IsBuiltinActor(c cid.Cid) bool {
	return builtin.IsBuiltinActor(c)
}

func IsAccountActor(c cid.Cid) bool {
	return c == builtin.AccountActorCodeID
}

func IsStorageMinerActor(c cid.Cid) bool {
	return c == builtin.StorageMinerActorCodeID
}

func IsMultisigActor(c cid.Cid) bool {
	return c == builtin.MultisigActorCodeID

}

func IsPaymentChannelActor(c cid.Cid) bool {
	return c == builtin.PaymentChannelActorCodeID
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
