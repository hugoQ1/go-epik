package miner

import (
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/cbor"
	"github.com/filecoin-project/go-state-types/dline"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
)

func init() {
	builtin.RegisterActorState(builtin2.StorageMinerActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load2(store, root)
	})
}

var Methods = builtin2.MethodsMiner

// Unchanged between v0 and v2 actors
var WPoStProvingPeriod = miner2.WPoStProvingPeriod
var WPoStPeriodDeadlines = miner2.WPoStPeriodDeadlines
var WPoStChallengeWindow = miner2.WPoStChallengeWindow
var WPoStChallengeLookback = miner2.WPoStChallengeLookback
var FaultDeclarationCutoff = miner2.FaultDeclarationCutoff

const MinSectorExpiration = miner2.MinSectorExpiration

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin2.StorageMinerActorCodeID:
		return load2(store, act.Head)
	}
	return nil, xerrors.Errorf("unknown actor code %s", act.Code)
}

type State interface {
	cbor.Marshaler

	// Total available balance to spend.
	AvailableBalance(abi.TokenAmount) (abi.TokenAmount, error)
	// Funds that will vest by the given epoch.
	VestedFunds(abi.ChainEpoch) (abi.TokenAmount, error)
	// Funds locked for various reasons.
	LockedFunds() (LockedFunds, error)
	FeeDebt() (abi.TokenAmount, error)

	GetSector(abi.SectorNumber) (*SectorOnChainInfo, error)
	FindSector(abi.SectorNumber) (*SectorLocation, error)
	GetSectorExpiration(abi.SectorNumber) (*SectorExpiration, error)
	GetPrecommittedSector(abi.SectorNumber) (*SectorPreCommitOnChainInfo, error)
	LoadSectors(sectorNos *bitfield.BitField) ([]*SectorOnChainInfo, error)
	NumLiveSectors() (uint64, error)
	IsAllocated(abi.SectorNumber) (bool, error)

	LoadDeadline(idx uint64) (Deadline, error)
	ForEachDeadline(cb func(idx uint64, dl Deadline) error) error
	NumDeadlines() (uint64, error)
	DeadlinesChanged(State) (bool, error)

	Info() (MinerInfo, error)
	MinerInfoChanged(State) (bool, error)

	DeadlineInfo(epoch abi.ChainEpoch) (*dline.Info, error)

	// Diff helpers. Used by Diff* functions internally.
	sectors() (adt.Array, error)
	decodeSectorOnChainInfo(*cbg.Deferred) (SectorOnChainInfo, error)
	precommits() (adt.Map, error)
	decodeSectorPreCommitOnChainInfo(*cbg.Deferred) (SectorPreCommitOnChainInfo, error)
}

type Deadline interface {
	LoadPartition(idx uint64) (Partition, error)
	ForEachPartition(cb func(idx uint64, part Partition) error) error
	PostSubmissions() (bitfield.BitField, error)

	PartitionsChanged(Deadline) (bool, error)
}

type Partition interface {
	AllSectors() (bitfield.BitField, error)
	FaultySectors() (bitfield.BitField, error)
	RecoveringSectors() (bitfield.BitField, error)
	LiveSectors() (bitfield.BitField, error)
	ActiveSectors() (bitfield.BitField, error)
}

type SectorOnChainInfo struct {
	SectorNumber          abi.SectorNumber
	SealProof             abi.RegisteredSealProof
	SealedCID             cid.Cid
	DealIDs               []abi.DealID
	Activation            abi.ChainEpoch
	Expiration            abi.ChainEpoch
	DealWeight            abi.DealWeight
	VerifiedDealWeight    abi.DealWeight
	InitialPledge         abi.TokenAmount
	ExpectedDayReward     abi.TokenAmount
	ExpectedStoragePledge abi.TokenAmount
}

type SectorPreCommitInfo = miner2.SectorPreCommitInfo

type SectorPreCommitOnChainInfo struct {
	Info               SectorPreCommitInfo
	PreCommitDeposit   abi.TokenAmount
	PreCommitEpoch     abi.ChainEpoch
	DealWeight         abi.DealWeight
	VerifiedDealWeight abi.DealWeight
}

type PoStPartition = miner2.PoStPartition
type RecoveryDeclaration = miner2.RecoveryDeclaration
type FaultDeclaration = miner2.FaultDeclaration

// Params
type DeclareFaultsParams = miner2.DeclareFaultsParams
type DeclareFaultsRecoveredParams = miner2.DeclareFaultsRecoveredParams
type SubmitWindowedPoStParams = miner2.SubmitWindowedPoStParams
type ProveCommitSectorParams = miner2.ProveCommitSectorParams

type MinerInfo struct {
	Owner                      address.Address   // Must be an ID-address.
	Worker                     address.Address   // Must be an ID-address.
	NewWorker                  address.Address   // Must be an ID-address.
	ControlAddresses           []address.Address // Must be an ID-addresses.
	WorkerChangeEpoch          abi.ChainEpoch
	PeerId                     *peer.ID
	Multiaddrs                 []abi.Multiaddrs
	SealProofType              abi.RegisteredSealProof
	SectorSize                 abi.SectorSize
	WindowPoStPartitionSectors uint64
	ConsensusFaultElapsed      abi.ChainEpoch
}

func (mi MinerInfo) IsController(addr address.Address) bool {
	if addr == mi.Owner || addr == mi.Worker {
		return true
	}

	for _, ca := range mi.ControlAddresses {
		if addr == ca {
			return true
		}
	}

	return false
}

type SectorExpiration struct {
	OnTime abi.ChainEpoch

	// non-zero if sector is faulty, epoch at which it will be permanently
	// removed if it doesn't recover
	Early abi.ChainEpoch
}

type SectorLocation struct {
	Deadline  uint64
	Partition uint64
}

type SectorChanges struct {
	Added    []SectorOnChainInfo
	Extended []SectorExtensions
	Removed  []SectorOnChainInfo
}

type SectorExtensions struct {
	From SectorOnChainInfo
	To   SectorOnChainInfo
}

type PreCommitChanges struct {
	Added   []SectorPreCommitOnChainInfo
	Removed []SectorPreCommitOnChainInfo
}

type LockedFunds struct {
	VestingFunds             abi.TokenAmount
	InitialPledgeRequirement abi.TokenAmount
	PreCommitDeposits        abi.TokenAmount
}

func (lf LockedFunds) TotalLockedFunds() abi.TokenAmount {
	return big.Add(lf.VestingFunds, big.Add(lf.InitialPledgeRequirement, lf.PreCommitDeposits))
}
