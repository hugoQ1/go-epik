package genesis

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"
)

type ActorType string

const (
	TAccount  ActorType = "account"
	TMultisig ActorType = "multisig"
)

type PreSeal struct {
	CommR     cid.Cid
	CommD     cid.Cid
	SectorID  abi.SectorNumber
	Deal      market2.DealProposal
	ProofType abi.RegisteredSealProof
}

type Miner struct {
	ID       address.Address
	Owner    address.Address
	Worker   address.Address
	Coinbase address.Address
	PeerId   peer.ID //nolint:golint

	MarketBalance abi.TokenAmount
	PowerBalance  abi.TokenAmount

	SectorSize abi.SectorSize

	Sectors []*PreSeal
}

type AccountMeta struct {
	Owner address.Address // bls / secpk
}

func (am *AccountMeta) ActorMeta() json.RawMessage {
	out, err := json.Marshal(am)
	if err != nil {
		panic(err)
	}
	return out
}

type MultisigMeta struct {
	Signers             []address.Address
	Threshold           int
	VestingDuration     int
	VestingStart        int
	InitialVestedTarget *builtin.BigFrac
}

func (mm *MultisigMeta) ActorMeta() json.RawMessage {
	out, err := json.Marshal(mm)
	if err != nil {
		panic(err)
	}
	return out
}

func (mm *MultisigMeta) InitialVestingBalance(total big.Int) big.Int {
	if total.LessThan(big.Zero()) {
		panic(fmt.Errorf("negative balance %v", total))
	}
	if mm.InitialVestedTarget == nil || mm.InitialVestedTarget.Denominator.IsZero() {
		return total
	}
	if mm.InitialVestedTarget.Numerator.LessThan(big.Zero()) ||
		mm.InitialVestedTarget.Denominator.LessThan(big.Zero()) ||
		mm.InitialVestedTarget.Denominator.LessThan(mm.InitialVestedTarget.Numerator) {
		panic(fmt.Errorf("illegal initial vested target num %v, den %v ", mm.InitialVestedTarget.Numerator, mm.InitialVestedTarget.Denominator))
	}
	initialVested := big.Div(big.Mul(total, mm.InitialVestedTarget.Numerator), mm.InitialVestedTarget.Denominator)
	if total.LessThan(initialVested) {
		panic(fmt.Errorf("initial vested %v less than total %v", initialVested, total))
	}
	return big.Sub(total, initialVested)
}

type Actor struct {
	Type    ActorType
	Balance abi.TokenAmount

	Meta json.RawMessage
}

type Template struct {
	Accounts []Actor
	Miners   []Miner

	NetworkName string
	Timestamp   uint64 `json:",omitempty"`

	// VerifregRootKey  Actor
	// RemainderAccount Actor
	TeamAccountActor          Actor
	FoundationAccountActor    Actor
	FundraisingAccountActor   Actor
	FirstGovernorAccountActor Actor
}
