// +build debug 2k devnet

package gen

import (
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/genesis"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

// team & contributors
var DefaultTeamAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromEpk(50_000_000), // 50M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("f3sjljpkhkgcosm5strgwnlwr5iuvbao54s6vhpeiqnyso75yioz6hdcrmm7xlccyacltjulivfiehb54ondxa"),
		},
		Threshold:       1,
		VestingDuration: 90 * 15 * builtin.EpochsInDay,
		VestingStart:    0,
		InitialVestedTarget: &builtin.BigFrac{
			Numerator:   big.NewInt(1),
			Denominator: big.NewInt(16),
		},
	}).ActorMeta(),
}

// foundation
var DefaultFoundationAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromEpk(50_000_000), // may be a little less than 50M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("f3vjp7dkpwx3prkjhyvl75az6qx6buw4kgzcvzopo7jydikhqellwfjl6n3v4hd5jiyp2sfg772zarv6vty42q"),
		},
		Threshold:       1,
		VestingDuration: 90 * 3 * builtin.EpochsInDay,
		VestingStart:    0,
		InitialVestedTarget: &builtin.BigFrac{
			Numerator:   big.NewInt(1),
			Denominator: big.NewInt(4),
		},
	}).ActorMeta(),
}

// investor
var DefaultInvestorAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromEpk(200_000_000), //  200M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("f3wn5zdvm2hkra3wtvtj5st7fweqjdg5e2h4epsujbemzmlq3hqlcfaktqqpcceemmtpqurw234kz2sy7v5zyq"),
		},
		Threshold:       1,
		VestingDuration: 90 * 6 * builtin.EpochsInDay,
		VestingStart:    0,
		InitialVestedTarget: &builtin.BigFrac{
			Numerator:   big.NewInt(1),
			Denominator: big.NewInt(7),
		},
	}).ActorMeta(),
}

//////////////////////
// 	default governor
//////////////////////
var DefaultGovernorActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: big.Zero(),
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("f3r3gyljdth7atl2lcsihuxr3e2ntdyuipdh24aszgwrbxc5ordtmkstw426kgq3bu7ubzrjtxoq4ote6vtdra"),
		},
		Threshold:       1,
		VestingDuration: 0,
		VestingStart:    0,
	}).ActorMeta(),
}

//////////////////////
// 	default expert
//////////////////////
var DefaultExpertActor = genesis.Actor{
	Type:    genesis.TAccount,
	Balance: abi.NewTokenAmount(0),
	Meta: (&genesis.AccountMeta{
		Owner: makeAddress("f3v37327k2kwfktufgpxspeqayne6cpfev4vzlb5wq555ehnyz3z3zkhzcvyh7vgikjzjpdf7f3wbzfkgyzapq"),
	}).ActorMeta(),
}

////////////////////////////////////
// 	default knowledge fund payee
////////////////////////////////////
var DefaultKgFundPayeeActor = genesis.Actor{
	Type:    genesis.TAccount,
	Balance: abi.NewTokenAmount(0),
	Meta: (&genesis.AccountMeta{
		Owner: makeAddress("f3qurflq6smm7fb7uhlswckysc6x52gaq3diwwkg7qdsoapzxrkliu2nddyzpevymqgwomwhtjltkfp4ndxzca"),
	}).ActorMeta(),
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
