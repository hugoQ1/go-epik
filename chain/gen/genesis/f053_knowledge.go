package genesis

import (
	"context"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	bstore "github.com/EpiK-Protocol/go-epik/lib/blockstore"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/util/adt"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/knowledge"
	cbor "github.com/ipfs/go-ipld-cbor"
)

func SetupKnowledgeActor(bs bstore.Blockstore, initPayee address.Address) (*types.Actor, error) {
	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))
	m, err := adt.MakeEmptyMap(store).Root()
	if err != nil {
		return nil, err
	}

	sms := knowledge.ConstructState(m, address.Undef)
	stcid, err := store.Put(store.Context(), sms)
	if err != nil {
		return nil, err
	}

	return &types.Actor{
		Code:    builtin.KnowledgeActorCodeID,
		Head:    stcid,
		Nonce:   0,
		Balance: types.NewInt(0),
	}, nil
}
