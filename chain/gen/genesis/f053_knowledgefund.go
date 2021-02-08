package genesis

import (
	"context"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	bstore "github.com/EpiK-Protocol/go-epik/lib/blockstore"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/knowledge"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
	"golang.org/x/xerrors"
)

func SetupKnowledgeActor(bs bstore.Blockstore, initialPayee address.Address) (*types.Actor, error) {
	if initialPayee.Protocol() != address.ID {
		return nil, xerrors.New("initial payee should be a ID-Address")
	}

	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))
	m, err := adt.MakeEmptyMap(store).Root()
	if err != nil {
		return nil, err
	}

	kas := knowledge.ConstructState(m, initialPayee)
	stcid, err := store.Put(store.Context(), kas)
	if err != nil {
		return nil, err
	}

	return &types.Actor{
		Code:    builtin.KnowledgeFundActorCodeID,
		Head:    stcid,
		Nonce:   0,
		Balance: types.NewInt(0),
	}, nil
}
