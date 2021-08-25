// +build devnet

package build

import (
	"os"

	"github.com/EpiK-Protocol/go-epik/chain/actors/policy"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

var DrandSchedule = map[abi.ChainEpoch]DrandEnum{
	0: DrandDevnet,
}

const BootstrappersFile = "devnet.pi"
const GenesisFile = "devnet.car"

func init() {
	policy.SetConsensusMinerMinPower(abi.NewStoragePower(1))

	if os.Getenv("EPIK_USE_TEST_ADDRESSES") != "1" {
		SetAddressNetwork(address.Mainnet)
	}

	Devnet = true

	BuildType = BuildDebug
}

const BlockDelaySecs = uint64(builtin2.EpochDurationSeconds)

const PropagationDelaySecs = uint64(6)

// BootstrapPeerThreshold is the minimum number peers we need to track for a sync worker to start
const BootstrapPeerThreshold = 1
