package genesis

import (
	"context"

	bstore "github.com/EpiK-Protocol/go-epik/blockstore"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/vote"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
)

func SetupVoteActor(bs bstore.Blockstore, fallback address.Address) (*types.Actor, error) {
	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))

	vas, err := vote.ConstructState(store, fallback)
	if err != nil {
		return nil, err
	}

	stcid, err := store.Put(store.Context(), vas)
	if err != nil {
		return nil, err
	}

	return &types.Actor{
		Code:    builtin.VoteFundActorCodeID,
		Head:    stcid,
		Nonce:   0,
		Balance: types.NewInt(0),
	}, nil
}
