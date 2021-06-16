package vesting

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
	builtin.RegisterActorState(builtin2.VestingActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load(store, root)
	})
}

var (
	Address = builtin2.VestingActorAddr
	Methods = builtin2.MethodsVesting
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.VestingActorCodeID:
		return load(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	TotalLocked() abi.TokenAmount
	Coinbase(address.Address, abi.ChainEpoch) (*CoinbaseInfo, error)
	TotalMined(address.Address) (abi.TokenAmount, error)
}

type CoinbaseInfo struct {
	Total   abi.TokenAmount
	Vested  abi.TokenAmount
	Vesting abi.TokenAmount
}
