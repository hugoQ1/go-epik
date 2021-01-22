package retrieval

import (
	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	retrieval2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/retrieval"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func init() {
	builtin.RegisterActorState(builtin2.RetrievalFundActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load2(store, root)
	})
}

var (
	Address = builtin2.RetrievalFundActorAddr
	Methods = builtin2.MethodsRetrieval
)

type RetrievalData = retrieval2.RetrievalDataParams
type WithdrawBalanceParams = retrieval2.WithdrawBalanceParams
type LockedState = retrieval2.LockedState

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.RetrievalFundActorCodeID:
		return load2(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	EscrowBalance(fromAddr address.Address) (abi.TokenAmount, error)
	DayExpend(epoch abi.ChainEpoch, fromAddr address.Address) (abi.TokenAmount, error)
	LockedState(fromAddr address.Address) (*LockedState, error)
	TotalCollateral() (abi.TokenAmount, error)
}
