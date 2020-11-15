package expert

import (
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	expert2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/expert"
)

func init() {
	builtin.RegisterActorState(builtin2.ExpertActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load2(store, root)
	})
}

var (
	Address = builtin2.ExpertFundsActorAddr
	Methods = builtin2.MethodsExpert
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.ExpertActorCodeID:
		return load2(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	Info() (ExpertInfo, error)
	Datas() ([]*DataOnChainInfo, error)
}

type ExpertInfo struct {
	Owner      address.Address // Must be an ID-address.
	PeerId     *peer.ID
	Multiaddrs []abi.Multiaddrs
}

type DataOnChainInfo = expert2.DataOnChainInfo
