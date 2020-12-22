package genesis

import (
	"context"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	bstore "github.com/EpiK-Protocol/go-epik/lib/blockstore"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
	"github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
)

func SetupExpertFundActor(bs bstore.Blockstore) (*types.Actor, error) {
	store := adt.WrapStore(context.TODO(), cbor.NewCborStore(bs))

	emptyMap, err := adt.MakeEmptyMap(store).Root()
	if err != nil {
		return nil, err
	}

	pool, err := store.Put(store.Context(), &expertfund.PoolInfo{
		LastRewardBlock: abi.ChainEpoch(0),
		AccPerShare:     abi.NewTokenAmount(0),
	})
	if err != nil {
		return nil, err
	}

	vas := expertfund.ConstructState(emptyMap, pool)
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
