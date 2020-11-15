package init

import (
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/node/modules/dtypes"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

func init() {
	builtin.RegisterActorState(builtin2.InitActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load2(store, root)
	})
}

var (
	Address = builtin2.InitActorAddr
	Methods = builtin2.MethodsInit
)

func Load(store adt.Store, act *types.Actor) (State, error) {
	switch act.Code {
	case builtin2.InitActorCodeID:
		return load2(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	ResolveAddress(address address.Address) (address.Address, bool, error)
	MapAddressToNewID(address address.Address) (address.Address, error)
	NetworkName() (dtypes.NetworkName, error)

	ForEachActor(func(id abi.ActorID, address address.Address) error) error

	// Remove exists to support tooling that manipulates state for testing.
	// It should not be used in production code, as init actor entries are
	// immutable.
	Remove(addrs ...address.Address) error

	// Sets the network's name. This should only be used on upgrade/fork.
	SetNetworkName(name string) error
}
