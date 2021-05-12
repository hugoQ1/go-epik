package expert

import (
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"

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

	Info() (*ExpertInfo, error)
	Datas() ([]*DataOnChainInfo, error)
	Data(cid.Cid) (*DataOnChainInfo, error)
}

type BatchImportDataParams = expert2.BatchImportDataParams
type ImportDataParams = expert2.ImportDataParams
type DataOnChainInfo = expert2.DataOnChainInfo

type ExpertInfo struct {
	expert2.ExpertInfo
	LostEpoch       abi.ChainEpoch
	Status          expert2.ExpertState
	StatusDesc      string // fill in state.go
	ImplicatedTimes uint64
	DataCount       uint64
	CurrentVotes    abi.TokenAmount // fill in state.go
	RequiredVotes   abi.TokenAmount
}
