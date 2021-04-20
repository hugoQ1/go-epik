package adt

import (
	"context"

	adt "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
	cbor "github.com/ipfs/go-ipld-cbor"
)

const DefaultMsgAmtBitwidth = 5

type Store interface {
	Context() context.Context
	cbor.IpldStore
}

func WrapStore(ctx context.Context, store cbor.IpldStore) Store {
	return adt.WrapStore(ctx, store)
}
