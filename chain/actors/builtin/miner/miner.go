package miner

import (
	"github.com/filecoin-project/go-state-types/network"
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

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	miner3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
)

func init() {
	builtin.RegisterActorState(builtin3.StorageMinerActorCodeID, func(store adt.Store, root cid.Cid) (cbor.Marshaler, error) {
		return load3(store, root)
	})
}

var Methods = builtin3.MethodsMiner

// Unchanged between v0, v2 and v3 actors
var WPoStProvingPeriod = miner3.WPoStProvingPeriod
var WPoStPeriodDeadlines = miner3.WPoStPeriodDeadlines
var WPoStChallengeWindow = miner3.WPoStChallengeWindow
var WPoStChallengeLookback = miner3.WPoStChallengeLookback
var FaultDeclarationCutoff = miner3.FaultDeclarationCutoff

// Not used / checked in v0
var DeclarationsMax = miner3.DeclarationsMax
var AddressedSectorsMax = miner3.AddressedSectorsMax

func Load(store adt.Store, act *types.Actor) (st State, err error) {
	switch act.Code {
	case builtin3.StorageMinerActorCodeID:
		return load3(store, act.Head)
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
	TotalPledge() (abi.TokenAmount, error)

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

	ContainsAnyPiece([]cid.Cid) (bool, error)

	// Diff helpers. Used by Diff* functions internally.
	sectors() (adt.Array, error)
	decodeSectorOnChainInfo(*cbg.Deferred) (SectorOnChainInfo, error)
	precommits() (adt.Map, error)
	decodeSectorPreCommitOnChainInfo(*cbg.Deferred) (SectorPreCommitOnChainInfo, error)
}

type Deadline interface {
	LoadPartition(idx uint64) (Partition, error)
	ForEachPartition(cb func(idx uint64, part Partition) error) error
	PartitionsPoSted() (bitfield.BitField, error)

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
	SectorNumber abi.SectorNumber
	SealProof    abi.RegisteredSealProof
	SealedCID    cid.Cid
	DealIDs      []abi.DealID
	Activation   abi.ChainEpoch
	PieceSizes   []abi.PaddedPieceSize
	DealWins     []builtin3.BoolValue
}

type SectorPreCommitInfo = miner3.SectorPreCommitInfo

type SectorPreCommitOnChainInfo struct {
	Info           SectorPreCommitInfo
	PreCommitEpoch abi.ChainEpoch
	PieceSizes     []abi.PaddedPieceSize
}

type PoStPartition = miner3.PoStPartition
type RecoveryDeclaration = miner3.RecoveryDeclaration
type FaultDeclaration = miner3.FaultDeclaration

// Params
type DeclareFaultsParams = miner3.DeclareFaultsParams
type DeclareFaultsRecoveredParams = miner3.DeclareFaultsRecoveredParams
type SubmitWindowedPoStParams = miner3.SubmitWindowedPoStParams
type ProveCommitSectorParams = miner3.ProveCommitSectorParams
type DisputeWindowedPoStParams = miner3.DisputeWindowedPoStParams

func PreferredSealProofTypeFromWindowPoStType(nver network.Version, proof abi.RegisteredPoStProof) (abi.RegisteredSealProof, error) {
	switch proof {
	case abi.RegisteredPoStProof_StackedDrgWindow2KiBV1:
		return abi.RegisteredSealProof_StackedDrg2KiBV1_1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow8MiBV1:
		return abi.RegisteredSealProof_StackedDrg8MiBV1_1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow512MiBV1:
		return abi.RegisteredSealProof_StackedDrg512MiBV1_1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow32GiBV1:
		return abi.RegisteredSealProof_StackedDrg32GiBV1_1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow64GiBV1:
		return abi.RegisteredSealProof_StackedDrg64GiBV1_1, nil
	default:
		return -1, xerrors.Errorf("unrecognized window post type: %d", proof)
	}
}

func WinningPoStProofTypeFromWindowPoStProofType(nver network.Version, proof abi.RegisteredPoStProof) (abi.RegisteredPoStProof, error) {
	switch proof {
	case abi.RegisteredPoStProof_StackedDrgWindow2KiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning2KiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow8MiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning8MiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow512MiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning512MiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow32GiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning32GiBV1, nil
	case abi.RegisteredPoStProof_StackedDrgWindow64GiBV1:
		return abi.RegisteredPoStProof_StackedDrgWinning64GiBV1, nil
	default:
		return -1, xerrors.Errorf("unknown proof type %d", proof)
	}
}

type MinerInfo struct {
	Owner                      address.Address   // Must be an ID-address.
	Worker                     address.Address   // Must be an ID-address.
	NewWorker                  address.Address   // Must be an ID-address.
	ControlAddresses           []address.Address // Must be an ID-addresses.
	Coinbase                   address.Address   // Must be an ID-address.
	WorkerChangeEpoch          abi.ChainEpoch
	PeerId                     *peer.ID
	Multiaddrs                 []abi.Multiaddrs
	WindowPoStProofType        abi.RegisteredPoStProof
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
	Added   []SectorOnChainInfo
	Removed []SectorOnChainInfo
	/* Extended []SectorExtensions */
}

/* type SectorExtensions struct {
	From SectorOnChainInfo
	To   SectorOnChainInfo
} */

type PreCommitChanges struct {
	Added   []SectorPreCommitOnChainInfo
	Removed []SectorPreCommitOnChainInfo
}

type LockedFunds struct {
	VestingFunds abi.TokenAmount
	/* InitialPledgeRequirement abi.TokenAmount
	PreCommitDeposits        abi.TokenAmount */
}

func (lf LockedFunds) TotalLockedFunds() abi.TokenAmount {
	/* return big.Add(lf.VestingFunds, big.Add(lf.InitialPledgeRequirement, lf.PreCommitDeposits)) */
	return lf.VestingFunds
}
