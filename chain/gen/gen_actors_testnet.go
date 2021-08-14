// +build testground

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
			makeAddress("f3vpgyu2hbcmchkjdzno2b7pxuekr6fnl4j6a7v6yjopzjctjqv54gbvdilwcfcf3ysemhacqt5mukkov7b3xa"),
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
			makeAddress("f3q7liklfxwhipm2d3vsvmyal5ern2fiqtuq6rpj3ocwb75yttjuggedd6qswivwsmvgfxbgw44iomfduhyxva"),
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
			makeAddress("f3sxpb7idf7oyame6wq3ol4qjnueygfugdoqw5p5lkwbwd7xru53nyhddqwwsyhh2ejzfkgmd5k5dmf2hqxvvq"),
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
			makeAddress("f3utrwamqldftbgsl6pwztlgpbqf546inkndlfctdo4od2uznzqcxpahssbnyaddeagplh5xmk6yfcwda7p44q"),
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
		Owner: makeAddress("f3sb4mkuv2ac3olqtvw4zh5zrtmgh6sgle7dgmpui4nxivvz6b7xn6qdz6xif7gmsz4h73jeark5y4hlngikrq"),
	}).ActorMeta(),
}

////////////////////////////////////
// 	default knowledge fund payee
////////////////////////////////////
var DefaultKgFundPayeeActor = genesis.Actor{
	Type:    genesis.TAccount,
	Balance: abi.NewTokenAmount(0),
	Meta: (&genesis.AccountMeta{
		Owner: makeAddress("f3srkibgeo6z5teaco5hjip4tt43cpgcs2adprem6vk5kd4vnzmtblyym2rqxzd2vkb2eg53m5ks6sj5xauiya"),
	}).ActorMeta(),
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
