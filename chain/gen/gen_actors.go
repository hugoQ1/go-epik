package gen

import (
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/genesis"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

// For testing
var (
	TestAddress = "t1dnas3yoc5bvz5evcuocb7tudimn2tpz63ajlk4y"
	TestPrivKey = "7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226e4a592b41555649724f596e6a51452f675653565274444f434374686d39785a4d7162764f7546794f69413d227d"
)

/////////////////
//	allocation
/////////////////

// team & contributors
var DefaultTeamAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromEpk(50_000_000), // 50M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("f3qmsny2ozauv2ltucxbmyfqnzbcypwq74dggmktxp6gnmji3dm63iqyyqssufii73lcu6au25rlnkvz6jvtea"),
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
			makeAddress("f3qmsny2ozauv2ltucxbmyfqnzbcypwq74dggmktxp6gnmji3dm63iqyyqssufii73lcu6au25rlnkvz6jvtea"),
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

// fundraising
var DefaultFundraisingAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromEpk(200_000_000), //  200M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("f3qmsny2ozauv2ltucxbmyfqnzbcypwq74dggmktxp6gnmji3dm63iqyyqssufii73lcu6au25rlnkvz6jvtea"),
		},
		Threshold:       1,
		VestingDuration: 90 * 7 * builtin.EpochsInDay,
		VestingStart:    0,
		InitialVestedTarget: &builtin.BigFrac{
			Numerator:   big.NewInt(1),
			Denominator: big.NewInt(8),
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
			makeAddress("f3qmsny2ozauv2ltucxbmyfqnzbcypwq74dggmktxp6gnmji3dm63iqyyqssufii73lcu6au25rlnkvz6jvtea"),
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
		Owner: makeAddress("f3qmsny2ozauv2ltucxbmyfqnzbcypwq74dggmktxp6gnmji3dm63iqyyqssufii73lcu6au25rlnkvz6jvtea"),
	}).ActorMeta(),
}

////////////////////////////////////
// 	default knowledge fund payee
////////////////////////////////////
var DefaultKgFundPayeeActor = genesis.Actor{
	Type:    genesis.TAccount,
	Balance: abi.NewTokenAmount(0),
	Meta: (&genesis.AccountMeta{
		Owner: makeAddress("f3qmsny2ozauv2ltucxbmyfqnzbcypwq74dggmktxp6gnmji3dm63iqyyqssufii73lcu6au25rlnkvz6jvtea"),
	}).ActorMeta(),
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
