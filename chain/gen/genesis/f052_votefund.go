package genesis

import (
	"context"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	bstore "github.com/EpiK-Protocol/go-epik/lib/blockstore"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/vote"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
)

func SetupVoteActor(bs bstore.Blockstore) (*types.Actor, error) {
	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))
	c, err := adt.MakeEmptyMap(store).Root()
	if err != nil {
		return nil, err
	}

	vas := vote.ConstructState(c)
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
