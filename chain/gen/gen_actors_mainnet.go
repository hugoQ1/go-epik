// +build !debug
// +build !2k
// +build !testground
// +build !calibnet
// +build !nerpanet
// +build !butterflynet
// +build !devnet

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
			makeAddress("f3qes5asp3flxi6w7ilwbnmdhqwcjdqxljbmuwof37dxq4q4nifdx52kitlnxe7nqohtpwp6euxth53loxlcia"),
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
			makeAddress("f3vd67sdu27mttcbs6j34uz3d77aizgbannm6bslt2fgu656sgeydxq27uw7r6vppfs6pvsx62gasowg4ssmqa"),
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
			makeAddress("f3xbgq6chq7g7xxfvl6ooazu54pl5ayiss3aelgyuewsqyirhpjh3a67ftizkg2xhwnkhfeqeq4zkpgkpj75lq"),
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
			makeAddress("f3romwhf6urwbxv73he7e5jj3spa2dszpr2y6ulbi5zdx65vqdk63zqxsbtzpy7g5eleyweo22neeqahiiub2a"),
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
		Owner: makeAddress("f3u4htuzaihfms2j2qcicx3kqlgzleve3qlj6ul6ei4iolc5ru6f4eof3vtv6sccanzymly7n5wv3znzijbedq"),
	}).ActorMeta(),
}

////////////////////////////////////
// 	default knowledge fund payee
////////////////////////////////////
var DefaultKgFundPayeeActor = genesis.Actor{
	Type:    genesis.TAccount,
	Balance: abi.NewTokenAmount(0),
	Meta: (&genesis.AccountMeta{
		Owner: makeAddress("f3xh2cf5httdpa4nsqo2fylag2uye5w3mmniem4gkjjfb7kjlhydnvw2caiz5oeq3hkx62u3vg5wgp5icnuopq"),
	}).ActorMeta(),
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
