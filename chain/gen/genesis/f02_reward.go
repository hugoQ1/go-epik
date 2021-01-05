package genesis

import (
	"context"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/reward"
	cbor "github.com/ipfs/go-ipld-cbor"

	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	bstore "github.com/EpiK-Protocol/go-epik/lib/blockstore"
)

func SetupRewardActor(bs bstore.Blockstore) (*types.Actor, error) {
	cst := cbor.NewCborStore(bs)

	st := reward.ConstructState()

	hcid, err := cst.Put(context.TODO(), st)
	if err != nil {
		return nil, err
	}

	return &types.Actor{
		Code:    builtin.RewardActorCodeID,
		Balance: types.BigInt{Int: build.InitialRewardBalance},
		Head:    hcid,
	}, nil
}
