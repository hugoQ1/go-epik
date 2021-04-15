package genesis

import (
	"context"

	bstore "github.com/EpiK-Protocol/go-epik/blockstore"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
)

func SetupExpertFundActor(bs bstore.Blockstore) (*types.Actor, error) {
	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))

	pool, err := store.Put(store.Context(), &expertfund.PoolInfo{
		TotalExpertDataSize: 0,
		AccPerShare:         abi.NewTokenAmount(0),
		LastRewardBalance:   abi.NewTokenAmount(0),
	})
	if err != nil {
		return nil, err
	}

	vas, err := expertfund.ConstructState(store, pool)
	if err != nil {
		return nil, err
	}

	stcid, err := store.Put(store.Context(), vas)
	if err != nil {
		return nil, err
	}

	return &types.Actor{
		Code:    builtin.ExpertFundActorCodeID,
		Head:    stcid,
		Nonce:   0,
		Balance: types.NewInt(0),
	}, nil
}
