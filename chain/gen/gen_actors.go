package gen

import (
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/genesis"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"
)

/////////////////
//	allocation
/////////////////

// team & contributors
var DefaultTeamAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromFil(50_000_000), // 50M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
		},
		Threshold:       3,
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
	Balance: types.FromFil(100_000_000), // may be a little less than 100M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
		},
		Threshold:       3,
		VestingDuration: 90 * 7 * builtin.EpochsInDay,
		VestingStart:    0,
		InitialVestedTarget: &builtin.BigFrac{
			Numerator:   big.NewInt(1),
			Denominator: big.NewInt(8),
		},
	}).ActorMeta(),
}

// fundraising
var DefaultFundraisingAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: types.FromFil(100_000_000), //  150M
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
		},
		Threshold:       3,
		VestingDuration: 90 * 7 * builtin.EpochsInDay,
		VestingStart:    0,
		InitialVestedTarget: &builtin.BigFrac{
			Numerator:   big.NewInt(1),
			Denominator: big.NewInt(8),
		},
	}).ActorMeta(),
}

/////////////////
// 	government
/////////////////
var DefaultGovernAccountActor = genesis.Actor{
	Type:    genesis.TMultisig,
	Balance: big.Zero(),
	Meta: (&genesis.MultisigMeta{
		Signers: []address.Address{
			makeAddress("t1ceb34gnsc6qk5dt6n7xg6ycwzasjhbxm3iylkiy"),
		},
		Threshold:       1,
		VestingDuration: 0,
		VestingStart:    0,
	}).ActorMeta(),
}

func makeAddress(addr string) address.Address {
	ret, err := address.NewFromString(addr)
	if err != nil {
		panic(err)
	}

	return ret
}
