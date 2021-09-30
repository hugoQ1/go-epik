package power

import (
	"bytes"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"

	builtin3 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	power3 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	adt3 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"
)

var _ State = (*state3)(nil)

func load3(store adt.Store, root cid.Cid) (State, error) {
	out := state3{store: store}
	err := store.Get(store.Context(), root, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type state3 struct {
	power3.State
	store adt.Store
}

func (s *state3) TotalLocked() (abi.TokenAmount, error) {
	return s.TotalPledgeCollateral, nil
}

func (s *state3) TotalPower() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalRawBytePower,
		QualityAdjPower: s.TotalQualityAdjPower,
	}, nil
}

// Committed power to the network. Includes miners below the minimum threshold.
func (s *state3) TotalCommitted() (Claim, error) {
	return Claim{
		RawBytePower:    s.TotalBytesCommitted,
		QualityAdjPower: s.TotalQABytesCommitted,
	}, nil
}

func (s *state3) PoStRatio() (out power3.WdPoStRatio, err error) {
	arr, err := adt3.AsArray(s.store, s.WdPoStRatios, builtin3.DefaultAmtBitwidth)
	if err != nil {
		return out, err
	}
	found, err := arr.Get(arr.Length()-1, &out)
	if err != nil {
		return out, err
	}
	if !found {
		return out, xerrors.Errorf("unexpected missing ratio at %d", arr.Length()-1)
	}
	return out, nil
}

func (s *state3) AllowNoPoSt(challenge abi.ChainEpoch, rand abi.Randomness) (bool, error) {
	return s.State.AllowNoPoSt(s.store, challenge, rand)
}

func (s *state3) MinerPower(addr address.Address) (Claim, bool, error) {
	claims, err := s.claims()
	if err != nil {
		return Claim{}, false, err
	}
	var claim power3.Claim
	ok, err := claims.Get(abi.AddrKey(addr), &claim)
	if err != nil {
		return Claim{}, false, err
	}
	if claim.TotalMiningPledge.LessThan(power3.ConsensusMinerMinPledge) {
		return Claim{
			RawBytePower:    big.Zero(),
			QualityAdjPower: big.Zero(),
		}, ok, nil
	}
	return Claim{
		RawBytePower:    claim.RawBytePower,
		QualityAdjPower: claim.QualityAdjPower,
	}, ok, nil
}

func (s *state3) MinerNominalPowerMeetsConsensusMinimum(miner address.Address, epoch abi.ChainEpoch, locked abi.TokenAmount) (bool, error) {
	return s.State.MinerNominalPowerMeetsConsensusMinimum(s.store, miner, epoch, locked)
}

func (s *state3) MinerCounts() (uint64, uint64, error) {
	return uint64(s.State.MinerAboveMinPowerCount), uint64(s.State.MinerCount), nil
}

func (s *state3) ListAllMiners() ([]address.Address, error) {
	claims, err := s.claims()
	if err != nil {
		return nil, err
	}

	var miners []address.Address
	err = claims.ForEach(nil, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		miners = append(miners, a)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return miners, nil
}

func (s *state3) ForEachClaim(cb func(miner address.Address, claim Claim) error) error {
	claims, err := s.claims()
	if err != nil {
		return err
	}

	var claim power3.Claim
	return claims.ForEach(&claim, func(k string) error {
		a, err := address.NewFromBytes([]byte(k))
		if err != nil {
			return err
		}
		if claim.TotalMiningPledge.LessThan(power3.ConsensusMinerMinPledge) {
			return cb(a, Claim{
				RawBytePower:    big.Zero(),
				QualityAdjPower: big.Zero(),
			})
		}
		return cb(a, Claim{
			RawBytePower:    claim.RawBytePower,
			QualityAdjPower: claim.QualityAdjPower,
		})
	})
}

func (s *state3) ClaimsChanged(other State) (bool, error) {
	other2, ok := other.(*state3)
	if !ok {
		// treat an upgrade as a change, always
		return true, nil
	}
	return !s.State.Claims.Equals(other2.State.Claims), nil
}

func (s *state3) claims() (adt.Map, error) {
	return adt3.AsMap(s.store, s.Claims, builtin3.DefaultHamtBitwidth)
}

func (s *state3) decodeClaim(val *cbg.Deferred) (Claim, error) {
	var ci power3.Claim
	if err := ci.UnmarshalCBOR(bytes.NewReader(val.Raw)); err != nil {
		return Claim{}, err
	}
	return fromV3Claim(ci), nil
}

func fromV3Claim(v3 power3.Claim) Claim {
	return Claim{
		RawBytePower:    v3.RawBytePower,
		QualityAdjPower: v3.QualityAdjPower,
	}
}

func (s *state3) PledgeReleasePeriod() (period abi.ChainEpoch, err error) {
	return s.State.MinerPledgeReleasePeriod, nil
}
