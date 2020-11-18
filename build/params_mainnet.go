// +build !debug
// +build !2k
// +build !testground

package build

import (
	"math"
	"os"

	"github.com/EpiK-Protocol/go-epik/chain/actors/policy"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

var DrandSchedule = map[abi.ChainEpoch]DrandEnum{
	0:                  DrandIncentinet,
	UpgradeSmokeHeight: DrandMainnet,
}

const UpgradeBreezeHeight = -1
const BreezeGasTampingDuration = 0

const UpgradeSmokeHeight = -1

const UpgradeIgnitionHeight = -2
const UpgradeRefuelHeight = -3

var UpgradeActorsV2Height = abi.ChainEpoch(0)

const UpgradeTapeHeight = -4

// This signals our tentative epoch for mainnet launch. Can make it later, but not earlier.
// Miners, clients, developers, custodians all need time to prepare.
// We still have upgrades and state changes to do, but can happen after signaling timing here.
const UpgradeLiftoffHeight = -5

const UpgradeKumquatHeight = -6

func init() {
	policy.SetConsensusMinerMinPower(abi.NewStoragePower(1))
	policy.SetSupportedProofTypes(
		abi.RegisteredSealProof_StackedDrg8MiBV1,
	)

	if os.Getenv("EPIK_USE_TEST_ADDRESSES") != "1" {
		SetAddressNetwork(address.Mainnet)
	}

	if os.Getenv("EPIK_DISABLE_V2_ACTOR_MIGRATION") == "1" {
		UpgradeActorsV2Height = math.MaxInt64
	}

	Devnet = false
}

const BlockDelaySecs = uint64(builtin2.EpochDurationSeconds)

const PropagationDelaySecs = uint64(6)

const BootstrapPeerThreshold = 4
