package genesis

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"

	market2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/market"

	"github.com/EpiK-Protocol/go-epik/chain/actors/adt"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/power"

	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/market"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/miner"

	"github.com/ipfs/go-cid"
	cbg "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"

	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	miner2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
	power2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/power"
	runtime2 "github.com/filecoin-project/specs-actors/v2/actors/runtime"

	"github.com/EpiK-Protocol/go-epik/chain/state"
	"github.com/EpiK-Protocol/go-epik/chain/store"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/chain/vm"
	"github.com/EpiK-Protocol/go-epik/genesis"
)

func MinerAddress(genesisIndex uint64) address.Address {
	maddr, err := address.NewIDAddress(MinerStart + genesisIndex)
	if err != nil {
		panic(err)
	}

	return maddr
}

type fakedSigSyscalls struct {
	runtime2.Syscalls
}

func (fss *fakedSigSyscalls) VerifySignature(signature crypto.Signature, signer address.Address, plaintext []byte) error {
	return nil
}

func mkFakedSigSyscalls(base vm.SyscallBuilder) vm.SyscallBuilder {
	return func(ctx context.Context, rt *vm.Runtime) runtime2.Syscalls {
		return &fakedSigSyscalls{
			base(ctx, rt),
		}
	}
}

func SetupStorageMiners(ctx context.Context, cs *store.ChainStore, sroot cid.Cid, tpl genesis.Template, inis InitDatas) (cid.Cid, error) {
	miners := tpl.Miners

	csc := func(context.Context, abi.ChainEpoch, *state.StateTree) (abi.TokenAmount, error) {
		return big.Zero(), nil
	}

	vmopt := &vm.VMOpts{
		StateBase:      sroot,
		Epoch:          0,
		Rand:           &fakeRand{},
		Bstore:         cs.Blockstore(),
		Syscalls:       mkFakedSigSyscalls(cs.VMSys()),
		CircSupplyCalc: csc,
		NtwkVersion:    genesisNetworkVersion,
		BaseFee:        types.NewInt(0),
	}

	vm, err := vm.NewVM(ctx, vmopt)
	if err != nil {
		return cid.Undef, xerrors.Errorf("failed to create NewVM: %w", err)
	}

	if len(miners) == 0 {
		return cid.Undef, xerrors.New("no genesis miners")
	}

	minerInfos := make([]struct {
		maddr   address.Address
		dealIDs []abi.DealID
	}, len(miners))
	for i, m := range miners {
		// Create miner through power actor
		i := i
		m := m

		{
			constructorParams := &power2.CreateMinerParams{
				Owner:               m.Worker,
				Worker:              m.Worker,
				Coinbase:            m.Coinbase,
				Peer:                []byte(m.PeerId),
				WindowPoStProofType: abi.RegisteredPoStProof_StackedDrgWindow8MiBV1,
			}

			params := mustEnc(constructorParams)
			rval, err := doExecValue(ctx, vm, power.Address, m.Owner, big.Zero(), builtin2.MethodsPower.CreateMiner, params)
			if err != nil {
				return cid.Undef, xerrors.Errorf("failed to create genesis miner: %w", err)
			}

			var ma power2.CreateMinerReturn
			if err := ma.UnmarshalCBOR(bytes.NewReader(rval)); err != nil {
				return cid.Undef, xerrors.Errorf("unmarshaling CreateMinerReturn: %w", err)
			}

			expma := MinerAddress(uint64(i) + 1) // plus 1 expert
			if ma.IDAddress != expma {
				return cid.Undef, xerrors.Errorf("miner assigned wrong address: %s != %s", ma.IDAddress, expma)
			}
			minerInfos[i].maddr = ma.IDAddress

			fmt.Printf("create genesis miner %s: Owner %s, Worker %s, Coinbase %s\n", ma.IDAddress, m.Owner, m.Worker, m.Coinbase)
		}

		// Add market funds

		if m.MarketBalance.GreaterThan(big.Zero()) {
			params := mustEnc(&minerInfos[i].maddr)
			_, err := doExecValue(ctx, vm, market.Address, m.Worker, m.MarketBalance, builtin2.MethodsMarket.AddBalance, params)
			if err != nil {
				return cid.Undef, xerrors.Errorf("failed to create genesis miner (add balance): %w", err)
			}
		}

		// Publish preseal deals

		{
			publish := func(params *market.PublishStorageDealsParams) error {
				fmt.Printf("publishing %d storage deals on miner %s with worker %s\n", len(params.Deals), params.Deals[0].Proposal.Provider, m.Worker)

				ret, err := doExecValue(ctx, vm, market.Address, m.Worker, big.Zero(), builtin2.MethodsMarket.PublishStorageDeals, mustEnc(params))
				if err != nil {
					return xerrors.Errorf("failed to create genesis miner (publish deals): %w", err)
				}
				var ids market.PublishStorageDealsReturn
				if err := ids.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
					return xerrors.Errorf("unmarsahling publishStorageDeals result: %w", err)
				}

				minerInfos[i].dealIDs = append(minerInfos[i].dealIDs, ids.IDs...)
				return nil
			}

			params := &market.PublishStorageDealsParams{
				DataRef: market2.PublishStorageDataRef{
					RootCID: inis.PresealPieceCID, // NOTE Piece CID
					Expert:  inis.Expert.String(),
				},
			}
			for _, preseal := range m.Sectors {
				params.Deals = append(params.Deals, market.ClientDealProposal{
					Proposal:        preseal.Deal,
					ClientSignature: crypto.Signature{Type: crypto.SigTypeBLS}, // TODO: do we want to sign these? Or do we want to fake signatures for genesis setup?
				})

				if len(params.Deals) == cbg.MaxLength {
					if err := publish(params); err != nil {
						return cid.Undef, err
					}

					params.Deals = []market2.ClientDealProposal{}
				}
			}

			if len(params.Deals) > 0 {
				if err := publish(params); err != nil {
					return cid.Undef, err
				}
			}
		}
	}

	/* // adjust total network power for equal pledge per sector
	rawPow, qaPow := big.NewInt(0), big.NewInt(0)
	{
		for i, m := range miners {
			for pi := range m.Sectors {
				rawPow = types.BigAdd(rawPow, types.NewInt(uint64(m.SectorSize)))

				dweight, err := dealWeight(ctx, vm, minerInfos[i].maddr, []abi.DealID{minerInfos[i].dealIDs[pi]}, 0, minerInfos[i].presealExp)
				if err != nil {
					return cid.Undef, xerrors.Errorf("getting deal weight: %w", err)
				}

				sectorWeight := miner2.QAPowerForWeight(m.SectorSize, minerInfos[i].presealExp, dweight.DealWeight, dweight.VerifiedDealWeight)

				qaPow = types.BigAdd(qaPow, sectorWeight)
			}
		}

		err = vm.MutateState(ctx, power.Address, func(cst cbor.IpldStore, st *power2.State) error {
			st.TotalQualityAdjPower = qaPow
			st.TotalRawBytePower = rawPow

			st.ThisEpochQualityAdjPower = qaPow
			st.ThisEpochRawBytePower = rawPow
			return nil
		})
		if err != nil {
			return cid.Undef, xerrors.Errorf("mutating state: %w", err)
		}

		err = vm.MutateState(ctx, reward.Address, func(sct cbor.IpldStore, st *reward2.State) error {
			*st = *reward2.ConstructState(qaPow)
			return nil
		})
		if err != nil {
			return cid.Undef, xerrors.Errorf("mutating state: %w", err)
		}
	} */

	rawPow := abi.NewStoragePower(0)
	qaPow := abi.NewStoragePower(0)
	for i, m := range miners {
		// Commit sectors
		{
			// add mining pledge
			_, err = doExecValue(ctx, vm, minerInfos[i].maddr, m.Worker, power2.ConsensusMinerMinPledge, builtin2.MethodsMiner.AddPledge, nil)
			if err != nil {
				return cid.Undef, xerrors.Errorf("failed to confirm presealed sectors (add pledge): %w", err)
			}

			for pi, preseal := range m.Sectors {
				params := &miner.SectorPreCommitInfo{
					SealProof:     preseal.ProofType,
					SectorNumber:  preseal.SectorID,
					SealedCID:     preseal.CommR,
					SealRandEpoch: -1,
					DealIDs:       []abi.DealID{minerInfos[i].dealIDs[pi]},
				}

				dealsInfo, err := dealWeight(ctx, vm, minerInfos[i].maddr, params.DealIDs, 0)
				if err != nil {
					return cid.Undef, xerrors.Errorf("getting deal weight: %w", err)
				}
				for _, size := range dealsInfo.Sectors[0].PieceSizes {
					qa := miner2.QAPowerForWeight(true, uint64(size))
					qaPow = big.Add(qaPow, qa)
					raw := miner2.QAPowerForWeight(false, uint64(size))
					rawPow = big.Add(rawPow, raw)
				}

				/* dweight, err := dealWeight(ctx, vm, minerInfos[i].maddr, params.DealIDs, 0, minerInfos[i].presealExp)
				if err != nil {
					return cid.Undef, xerrors.Errorf("getting deal weight: %w", err)
				}

				sectorWeight := miner2.QAPowerForWeight(m.SectorSize, minerInfos[i].presealExp, dweight.DealWeight, dweight.VerifiedDealWeight)

				// we've added fake power for this sector above, remove it now
				err = vm.MutateState(ctx, power.Address, func(cst cbor.IpldStore, st *power2.State) error {
					st.TotalQualityAdjPower = types.BigSub(st.TotalQualityAdjPower, sectorWeight) //nolint:scopelint
					st.TotalRawBytePower = types.BigSub(st.TotalRawBytePower, types.NewInt(uint64(m.SectorSize)))
					return nil
				})
				if err != nil {
					return cid.Undef, xerrors.Errorf("removing fake power: %w", err)
				} */

				/* epochReward, err := currentEpochBlockReward(ctx, vm, minerInfos[i].maddr)
				if err != nil {
					return cid.Undef, xerrors.Errorf("getting current epoch reward: %w", err)
				}

				tpow, err := currentTotalPower(ctx, vm, minerInfos[i].maddr)
				if err != nil {
					return cid.Undef, xerrors.Errorf("getting current total power: %w", err)
				}

				pcd := miner2.PreCommitDepositForPower(epochReward.ThisEpochRewardSmoothed, tpow.QualityAdjPowerSmoothed, sectorWeight)

				pledge := miner2.InitialPledgeForPower(
					sectorWeight,
					epochReward.ThisEpochBaselinePower,
					epochReward.ThisEpochRewardSmoothed,
					tpow.QualityAdjPowerSmoothed,
					circSupply(ctx, vm, minerInfos[i].maddr),
				)

				pledge = big.Add(pcd, pledge)

				fmt.Println(types.EPK(pledge)) */
				_, err = doExecValue(ctx, vm, minerInfos[i].maddr, m.Worker, big.Zero(), builtin2.MethodsMiner.PreCommitSector, mustEnc(params))
				if err != nil {
					return cid.Undef, xerrors.Errorf("failed to confirm presealed sectors (pre-commit): %w", err)
				}

				// Commit one-by-one, otherwise pledge math tends to explode
				confirmParams := &builtin2.ConfirmSectorProofsParams{
					Sectors: []abi.SectorNumber{preseal.SectorID},
				}

				_, err = doExecValue(ctx, vm, minerInfos[i].maddr, power.Address, big.Zero(), builtin2.MethodsMiner.ConfirmSectorProofsValid, mustEnc(confirmParams))
				if err != nil {
					return cid.Undef, xerrors.Errorf("failed to confirm presealed sectors (confirm proof): %w", err)
				}
			}
		}
	}

	// TODO: Should we re-ConstructState for the reward actor using rawPow as currRealizedPower here?

	c, err := vm.Flush(ctx)
	if err != nil {
		return cid.Undef, xerrors.Errorf("flushing vm: %w", err)
	}

	err = checkPowerActor(vm, cs.Store(ctx), rawPow, qaPow, big.Mul(power2.ConsensusMinerMinPledge, big.NewInt(int64(len(minerInfos)))))
	if err != nil {
		return cid.Undef, xerrors.Errorf("check genesis power state: %w", err)
	}

	err = checkMarketActor(vm, cs.Store(ctx), inis.PresealPieceCID, int64(len(minerInfos)))
	if err != nil {
		return cid.Undef, xerrors.Errorf("check genesis market state: %w", err)
	}

	return c, nil
}

// TODO: copied from actors test harness, deduplicate or remove from here
type fakeRand struct{}

func (fr *fakeRand) GetChainRandomness(ctx context.Context, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) ([]byte, error) {
	out := make([]byte, 32)
	_, _ = rand.New(rand.NewSource(int64(randEpoch * 1000))).Read(out) //nolint
	return out, nil
}

func (fr *fakeRand) GetBeaconRandomness(ctx context.Context, personalization crypto.DomainSeparationTag, randEpoch abi.ChainEpoch, entropy []byte) ([]byte, error) {
	out := make([]byte, 32)
	_, _ = rand.New(rand.NewSource(int64(randEpoch))).Read(out) //nolint
	return out, nil
}

func checkPowerActor(vm *vm.VM, stor adt.Store, expectRawPower, expectQAPower abi.StoragePower, expectPledge abi.TokenAmount) error {
	act, err := vm.StateTree().GetActor(power.Address)
	if err != nil {
		return xerrors.Errorf("failed to get power actor %s: %w", power.Address, err)
	}
	st, err := power.Load(stor, act)
	if err != nil {
		return xerrors.Errorf("failed to load power state: %w", err)
	}
	claim, err := st.TotalPower()
	if err != nil {
		return xerrors.Errorf("failed to get total power: %w", err)
	}
	if !claim.RawBytePower.Equals(expectRawPower) {
		return xerrors.Errorf("TotalRawBytePower %s doesn't match previously calculated rawPow %s", claim.RawBytePower, expectRawPower)
	}

	if !claim.QualityAdjPower.Equals(expectQAPower) {
		return xerrors.Errorf("TotalQualityAdjPower %s doesn't match previously calculated qaPow %s", claim.QualityAdjPower, expectQAPower)
	}

	locked, err := st.TotalLocked()
	if err != nil {
		return xerrors.Errorf("failed to get total locked: %w", err)
	}
	if !locked.Equals(expectPledge) {
		return xerrors.Errorf("TotalPledgeCollateral %s doesn't match expected %s", locked, expectPledge)
	}

	return nil
}

func checkMarketActor(vm *vm.VM, stor adt.Store, pieceCID cid.Cid, expectQuotaAcquired int64) error {
	act, err := vm.StateTree().GetActor(market.Address)
	if err != nil {
		return xerrors.Errorf("failed to get market actor %s: %w", market.Address, err)
	}
	st, err := market.Load(stor, act)
	if err != nil {
		return xerrors.Errorf("failed to load market state: %w", err)
	}
	qts, err := st.Quotas()
	if err != nil {
		return err
	}
	rm, err := qts.RemainingQuota(pieceCID)
	if err != nil {
		return xerrors.Errorf("failed to get remaining quota for piece %s: %w", pieceCID, err)
	}
	if rm+expectQuotaAcquired != market2.DefaultInitialQuota {
		return xerrors.Errorf("Remaining genesis file quota %d doesn't match expected %d", rm, market2.DefaultInitialQuota-expectQuotaAcquired)
	}
	return nil
}

/* func currentTotalPower(ctx context.Context, vm *vm.VM, maddr address.Address) (*power2.CurrentTotalPowerReturn, error) {
	pwret, err := doExecValue(ctx, vm, power.Address, maddr, big.Zero(), builtin2.MethodsPower.CurrentTotalPower, nil)
	if err != nil {
		return nil, err
	}
	var pwr power2.CurrentTotalPowerReturn
	if err := pwr.UnmarshalCBOR(bytes.NewReader(pwret)); err != nil {
		return nil, err
	}

	return &pwr, nil
} */

func dealWeight(ctx context.Context, vm *vm.VM, maddr address.Address, dealIDs []abi.DealID, sectorStart /* , sectorExpiry  */ abi.ChainEpoch) (market2.VerifyDealsForActivationReturn, error) {
	params := &market2.VerifyDealsForActivationParams{
		Sectors: []market2.SectorDeals{{
			DealIDs: dealIDs,
		}},
	}

	var dealWeights market2.VerifyDealsForActivationReturn
	ret, err := doExecValue(ctx, vm,
		market.Address,
		maddr,
		abi.NewTokenAmount(0),
		builtin2.MethodsMarket.VerifyDealsForActivation,
		mustEnc(params),
	)
	if err != nil {
		return market2.VerifyDealsForActivationReturn{}, err
	}
	if err := dealWeights.UnmarshalCBOR(bytes.NewReader(ret)); err != nil {
		return market2.VerifyDealsForActivationReturn{}, err
	}

	return dealWeights, nil
}

/* func currentEpochBlockReward(ctx context.Context, vm *vm.VM, maddr address.Address) (*reward2.ThisEpochRewardReturn, error) {
	rwret, err := doExecValue(ctx, vm, reward.Address, maddr, big.Zero(), builtin2.MethodsReward.ThisEpochReward, nil)
	if err != nil {
		return nil, err
	}

	var epochReward reward2.ThisEpochRewardReturn
	if err := epochReward.UnmarshalCBOR(bytes.NewReader(rwret)); err != nil {
		return nil, err
	}

	return &epochReward, nil
}

func circSupply(ctx context.Context, vmi *vm.VM, maddr address.Address) abi.TokenAmount {
	unsafeVM := &vm.UnsafeVM{VM: vmi}
	rt := unsafeVM.MakeRuntime(ctx, &types.Message{
		GasLimit: 1_000_000_000,
		From:     maddr,
	})

	return rt.TotalFilCircSupply()
} */
