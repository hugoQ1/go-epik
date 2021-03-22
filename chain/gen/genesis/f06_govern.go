package genesis

import (
	"context"

	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/govern"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	bstore "github.com/EpiK-Protocol/go-epik/blockstore"
	"github.com/EpiK-Protocol/go-epik/chain/types"
)

func SetupGovernActor(bs bstore.Blockstore, super address.Address) (*types.Actor, error) {
	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))
	sms, err := govern.ConstructState(store, super)
	if err != nil {
		return nil, err
	}

	stcid, err := store.Put(store.Context(), sms)
	if err != nil {
		return nil, err
	}

	act := &types.Actor{
		Code:    builtin.GovernActorCodeID,
		Head:    stcid,
		Balance: types.NewInt(0),
	}

	return act, nil
}
