// +build !testground

package build

import (
	"math/big"
	"os"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/EpiK-Protocol/go-epik/chain/actors/policy"
)

// /////
// Storage

const UnixfsChunkSize uint64 = 1 << 20
const UnixfsLinksPerLevel = 1024

// /////
// Consensus / Network

const AllowableClockDriftSecs = uint64(1)
const NewestNetworkVersion = network.Version8

// const ActorUpgradeNetworkVersion = network.Version4

// Epochs
const ForkLengthThreshold = Finality

// Blocks (e)
var BlocksPerEpoch = uint64(builtin2.ExpectedLeadersPerEpoch)

// Epochs
const Finality = policy.ChainFinality
const MessageConfidence = uint64(5)

// constants for Weight calculation
// The ratio of weight contributed by short-term vs long-term factors in a given round
const WRatioNum = int64(1)
const WRatioDen = uint64(2)

// /////
// Proofs

// Epochs
// TODO: unused
const SealRandomnessLookback = policy.SealRandomnessLookback

// /////
// Mining

// Epochs
const TicketRandomnessLookback = abi.ChainEpoch(1)

// /////
// Address

const AddressMainnetEnvVar = "_mainnet_"

// the 'f' prefix doesn't matter
var ZeroAddress = MustParseAddress("f3yaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaby2smx7a")

// /////
// Devnet settings

var Devnet = true

const EpkBase = uint64(1_000_000_000)
const EpkAllocStorageMining = uint64(700_000_000)

const EpkPrecision = uint64(1_000_000_000_000_000_000)
const EpkReserved = uint64(300_000_000)

var InitialRewardBalance *big.Int
var InitialEpkReserved *big.Int

// TODO: Move other important consts here

func init() {
	InitialRewardBalance = big.NewInt(int64(EpkAllocStorageMining))
	InitialRewardBalance = InitialRewardBalance.Mul(InitialRewardBalance, big.NewInt(int64(EpkPrecision)))

	InitialEpkReserved = big.NewInt(int64(EpkReserved))
	InitialEpkReserved = InitialEpkReserved.Mul(InitialEpkReserved, big.NewInt(int64(EpkPrecision)))

	if os.Getenv("EPIK_ADDRESS_TYPE") == AddressMainnetEnvVar {
		SetAddressNetwork(address.Mainnet)
	}
}

// Sync
const BadBlockCacheSize = 1 << 15

// assuming 4000 messages per round, this lets us not lose any messages across a
// 10 block reorg.
const BlsSignatureCacheSize = 40000

// Size of signature verification cache
// 32k keeps the cache around 10MB in size, max
const VerifSigCacheSize = 32000

// ///////
// Limits

// TODO: If this is gonna stay, it should move to specs-actors
const BlockMessageLimit = 10000

const BlockGasLimit = 10_000_000_000
const BlockGasTarget = BlockGasLimit / 2
const BaseFeeMaxChangeDenom = 8 // 12.5%
const InitialBaseFee = 100e6
const MinimumBaseFee = 100
const PackingEfficiencyNum = 4
const PackingEfficiencyDenom = 5

/* // Actor consts
var MinDealDuration = abi.ChainEpoch(180 * builtin2.EpochsInDay) */
