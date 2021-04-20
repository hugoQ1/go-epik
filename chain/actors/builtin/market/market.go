package market

import (
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	market3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"
)

func init() {
	// builtin.RegisterActorState(builtin2.StorageMarketActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
	// 	return load2(store, root)
	// })
	builtin.RegisterActorState(builtin3.StorageMarketActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load3(store, root)
	})
}

var (
	Address = builtin3.StorageMarketActorAddr
	Methods = builtin3.MethodsMarket
)

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin3.StorageMarketActorCodeID:
		return load3(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler
	BalancesChanged(State) (bool, error)
	EscrowTable() (BalanceTable, error)
	LockedTable() (BalanceTable, error)
	TotalLocked() (abi.TokenAmount, error)
	StatesChanged(State) (bool, error)
	States() (DealStates, error)
	ProposalsChanged(State) (bool, error)
	Proposals() (DealProposals, error)
	Quotas() (Quotas, error)
	DataIndexes() (DataIndexes, error)
	HasPendingPiece(address.Address, []cid.Cid) (bool, error)
}

type BalanceTable interface {
	ForEach(cb func(address.Address, abi.TokenAmount) error) error
	Get(key address.Address) (abi.TokenAmount, error)
}

type DealStates interface {
	ForEach(cb func(id abi.DealID, ds DealState) error) error
	Get(id abi.DealID) (*DealState, bool, error)

	array() adt.Array
	decode(*cbg.Deferred) (*DealState, error)
}

type DataIndexes interface {
	ForEach(epoch abi.ChainEpoch, cb func(provider address.Address, index DataIndex) error) error
}

type DealProposals interface {
	ForEach(cb func(id abi.DealID, dp DealProposal) error) error
	Get(id abi.DealID) (*DealProposal, bool, error)

	array() adt.Array
	decode(*cbg.Deferred) (*DealProposal, error)
}

type Quotas interface {
	InitialQuota() int64
	RemainingQuota(pieceCID cid.Cid) (int64, error)
}

type StorageDataRef = market3.StorageDataRef
type PublishStorageDealsParams = market3.PublishStorageDealsParams
type PublishStorageDealsReturn = market3.PublishStorageDealsReturn
type VerifyDealsForActivationParams = market3.VerifyDealsForActivationParams
type WithdrawBalanceParams = market3.WithdrawBalanceParams

type ClientDealProposal = market3.ClientDealProposal
type DataIndex = market3.DataIndex

type DealState struct {
	SectorStartEpoch abi.ChainEpoch // -1 if not yet included in proven sector
	LastUpdatedEpoch abi.ChainEpoch // -1 if deal state never updated
	SlashEpoch       abi.ChainEpoch // -1 if deal never slashed
}

type DealProposal struct {
	PieceCID  cid.Cid
	PieceSize abi.PaddedPieceSize
	/* VerifiedDeal         bool */
	Client     address.Address
	Provider   address.Address
	Label      string
	StartEpoch abi.ChainEpoch
	/* EndEpoch             abi.ChainEpoch
	StoragePricePerEpoch abi.TokenAmount
	ProviderCollateral   abi.TokenAmount
	ClientCollateral     abi.TokenAmount */
}

type DealStateChanges struct {
	Added    []DealIDState
	Modified []DealStateChange
	Removed  []DealIDState
}

type DealIDState struct {
	ID   abi.DealID
	Deal DealState
}

// DealStateChange is a change in deal state from -> to
type DealStateChange struct {
	ID   abi.DealID
	From *DealState
	To   *DealState
}

type DealProposalChanges struct {
	Added   []ProposalIDState
	Removed []ProposalIDState
}

type ProposalIDState struct {
	ID       abi.DealID
	Proposal DealProposal
}

func EmptyDealState() *DealState {
	return &DealState{
		SectorStartEpoch: -1,
		SlashEpoch:       -1,
		LastUpdatedEpoch: -1,
	}
}
