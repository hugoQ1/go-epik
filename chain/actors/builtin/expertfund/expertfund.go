package expertfund

import (
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/cbor"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
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
}