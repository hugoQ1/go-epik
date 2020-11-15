package power

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

func init() {
	builtin.RegisterActorState(builtin2.StoragePowerActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load2(store, root)
	})
}

var (
	Address = builtin2.StoragePowerActorAddr
	Methods = builtin2.MethodsPower
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.StoragePowerActorCodeID:
		return load2(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	TotalLocked() (abi.TokenAmount, error)
	TotalPower() (Claim, error)
	TotalCommitted() (Claim, error)
	TotalPowerSmoothed() (builtin.FilterEstimate, error)

	// MinerCounts returns the number of miners. Participating is the number
	// with power above the minimum miner threshold.
	MinerCounts() (participating, total uint64, err error)
	MinerPower(address.Address) (Claim, bool, error)
	MinerNominalPowerMeetsConsensusMinimum(address.Address) (bool, error)
	ListAllMiners() ([]address.Address, error)
	ListAllExperts() ([]address.Address, error)
	ForEachClaim(func(miner address.Address, claim Claim) error) error
	ClaimsChanged(State) (bool, error)

	// Diff helpers. Used by Diff* functions internally.
	claims() (adt.Map, error)
	decodeClaim(*cbg.Deferred) (Claim, error)
}

type Claim struct {
	// Sum of raw byte power for a miner's sectors.
	RawBytePower abi.StoragePower

	// Sum of quality adjusted power for a miner's sectors.
	QualityAdjPower abi.StoragePower
}

func AddClaims(a Claim, b Claim) Claim {
	return Claim{
		RawBytePower:    big.Add(a.RawBytePower, b.RawBytePower),
		QualityAdjPower: big.Add(a.QualityAdjPower, b.QualityAdjPower),
	}
}
