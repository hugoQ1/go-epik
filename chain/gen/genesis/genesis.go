package genesis

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/journal"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	cbor "github.com/ipfs/go-ipld-cbor"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	builtin2 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	account2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/account"
	expert2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/expert"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/expertfund"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/govern"
	multisig2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	adt2 "github.com/filecoin-project/specs-actors/v2/actors/util/adt"

	bstore "github.com/EpiK-Protocol/go-epik/blockstore"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/state"
	"github.com/EpiK-Protocol/go-epik/chain/store"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/chain/vm"
	"github.com/EpiK-Protocol/go-epik/genesis"
)

const AccountStart = 100
const MinerStart = 1000
const MaxAccounts = MinerStart - AccountStart

var log = logging.Logger("genesis")

type GenesisBootstrap struct {
	Genesis *types.BlockHeader
}

/*
From a list of parameters, create a genesis block / initial state

The process:
- Bootstrap state (MakeInitialStateTree)
  - Create empty state
  - Create system actor
  - Make init actor
    - Create accounts mappings
    - Set NextID to MinerStart
  - Setup Reward (0.7B epk)
  - Setup Cron
  - Create empty power actor
  - Create empty market actor
  - Create empty govern actor
  - Setup burnt fund address
  - Setup expert funds actor
  - Setup retrieval funds actor
  - Setup vote funds actor
  - Setup knowledge funds actor
  - Initialize account / msig balances
- Instantiate early vm with genesis syscalls
  - Create miners
    - Each:
      - power.CreateMiner, set msg value to PowerBalance
      - market.AddFunds with correct value
      - market.PublishDeals for related sectors
    - Set network power in the power actor to what we'll have after genesis creation
	- Recreate reward actor state with the right power
    - For each precommitted sector
      - Get deal weight
      - Calculate QA Power
      - Remove fake power from the power actor
      - Calculate pledge
      - Precommit
      - Confirm valid

Data Types:

PreSeal :{
  CommR    CID
  CommD    CID
  SectorID SectorNumber
  Deal     market.DealProposal # Start at 0, self-deal!
}

Genesis: {
	Accounts: [ # non-miner, non-singleton actors, max len = MaxAccounts
		{
			Type: "account" / "multisig",
			Value: "attoepk",
			[Meta: {msig settings, account key..}]
		},...
	],
	Miners: [
		{
			Owner, Worker Addr # ID
			MarketBalance, PowerBalance TokenAmount
			SectorSize uint64
			PreSeals []PreSeal
		},...
	],
}

*/

func MakeInitialStateTree(ctx context.Context, bs bstore.Blockstore, template genesis.Template) (*state.StateTree, map[address.Address]address.Address, error) {
	// Create empty state tree

	cst := cbor.NewCborStore(bs)
	_, err := cst.Put(context.TODO(), []struct{}{})
	if err != nil {
		return nil, nil, xerrors.Errorf("putting empty object: %w", err)
	}

	state, err := state.NewStateTree(cst, types.StateTreeVersion2)
	if err != nil {
		return nil, nil, xerrors.Errorf("making new state tree: %w", err)
	}

	// Create system actor

	sysact, err := SetupSystemActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup init actor: %w", err)
	}
	if err := state.SetActor(builtin2.SystemActorAddr, sysact); err != nil {
		return nil, nil, xerrors.Errorf("set init actor: %w", err)
	}

	// Create init actor

	idStart, initact, keyIDs, err := SetupInitActor(bs, template)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup init actor: %w", err)
	}
	if err := state.SetActor(builtin2.InitActorAddr, initact); err != nil {
		return nil, nil, xerrors.Errorf("set init actor: %w", err)
	}

	// Setup reward
	// RewardActor's state is overrwritten by SetupStorageMiners
	rewact, err := SetupRewardActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup init actor: %w", err)
	}

	err = state.SetActor(builtin2.RewardActorAddr, rewact)
	if err != nil {
		return nil, nil, xerrors.Errorf("set network account actor: %w", err)
	}

	// Setup cron
	cronact, err := SetupCronActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup cron actor: %w", err)
	}
	if err := state.SetActor(builtin2.CronActorAddr, cronact); err != nil {
		return nil, nil, xerrors.Errorf("set cron actor: %w", err)
	}

	// Create empty power actor
	spact, err := SetupStoragePowerActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup storage power actor: %w", err)
	}
	if err := state.SetActor(builtin2.StoragePowerActorAddr, spact); err != nil {
		return nil, nil, xerrors.Errorf("set storage market actor: %w", err)
	}

	// Create empty market actor
	marketact, err := SetupStorageMarketActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup storage market actor: %w", err)
	}
	if err := state.SetActor(builtin2.StorageMarketActorAddr, marketact); err != nil {
		return nil, nil, xerrors.Errorf("set market actor: %w", err)
	}

	// Create govern actor
	governact, err := SetupGovernActor(bs, builtin.FoundationIDAddress)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup govern actor: %w", err)
	}
	if err := state.SetActor(builtin2.GovernActorAddr, governact); err != nil {
		return nil, nil, xerrors.Errorf("set govern actor: %w", err)
	}

	// Setup burnt-funds
	burntRoot, err := cst.Put(ctx, &account2.State{
		Address: builtin2.BurntFundsActorAddr,
	})
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to setup burnt funds actor state: %w", err)
	}
	err = state.SetActor(builtin2.BurntFundsActorAddr, &types.Actor{
		Code:    builtin2.AccountActorCodeID,
		Balance: types.NewInt(0),
		Head:    burntRoot,
	})
	if err != nil {
		return nil, nil, xerrors.Errorf("set burnt funds account actor: %w", err)
	}

	// Create expert fund actor
	expertfundact, err := SetupExpertFundActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup expert fund actor: %w", err)
	}
	if err := state.SetActor(builtin2.ExpertFundActorAddr, expertfundact); err != nil {
		return nil, nil, xerrors.Errorf("set expert fund actor: %w", err)
	}

	// Create retrieve fund actor
	retrievalact, err := SetupRetrievalFundActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup retrieval fund actor: %w", err)
	}
	if err := state.SetActor(builtin2.RetrievalFundActorAddr, retrievalact); err != nil {
		return nil, nil, xerrors.Errorf("set retrieval fund actor: %w", err)
	}

	// Create vote fund actor
	voteact, err := SetupVoteActor(bs, builtin.FoundationIDAddress)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup vote actor: %w", err)
	}
	if err := state.SetActor(builtin2.VoteFundActorAddr, voteact); err != nil {
		return nil, nil, xerrors.Errorf("set vote actor: %w", err)
	}

	// Create accounts
	for _, info := range template.Accounts {

		switch info.Type {
		case genesis.TAccount:
			if _, err := createAccountActor(ctx, cst, state, info, keyIDs); err != nil {
				return nil, nil, xerrors.Errorf("failed to create account actor: %w", err)
			}

		case genesis.TMultisig:

			ida, err := address.NewIDAddress(uint64(idStart))
			if err != nil {
				return nil, nil, err
			}
			idStart++

			if err := createMultisigAccount(ctx, bs, cst, state, ida, info, keyIDs); err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, xerrors.New("unsupported account type")
		}

	}

	// Create default expert actor
	if _, err = createAccountActor(ctx, cst, state, template.DefaultExpertActor, keyIDs); err != nil {
		return nil, nil, xerrors.Errorf("failed to create default expert actor: %w", err)
	}

	// Create knowledge fund payee
	var kgFundPayeeIDAddr address.Address
	switch template.DefaultKgFundPayeeActor.Type {
	case genesis.TAccount:
		if kgFundPayeeIDAddr, err = createAccountActor(ctx, cst, state, template.DefaultKgFundPayeeActor, keyIDs); err != nil {
			return nil, nil, xerrors.Errorf("failed to create payee account actor: %w", err)
		}

	case genesis.TMultisig:
		kgFundPayeeIDAddr, err = address.NewIDAddress(uint64(idStart))
		if err != nil {
			return nil, nil, err
		}
		idStart++

		if err := createMultisigAccount(ctx, bs, cst, state, kgFundPayeeIDAddr, template.DefaultKgFundPayeeActor, keyIDs); err != nil {
			return nil, nil, err
		}

	default:
		return nil, nil, xerrors.New("unsupported knowledge fund payee type")
	}

	// Create knowledge fund actor
	knowact, err := SetupKnowledgeActor(bs, kgFundPayeeIDAddr)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup knowledge actor: %w", err)
	}
	if err := state.SetActor(builtin2.KnowledgeFundActorAddr, knowact); err != nil {
		return nil, nil, xerrors.Errorf("set knowledge actor: %w", err)
	}

	// Create vesting actor
	vestingact, err := SetupVestingActor(bs)
	if err != nil {
		return nil, nil, xerrors.Errorf("setup vesting actor: %w", err)
	}
	if err := state.SetActor(builtin2.VestingActorAddr, vestingact); err != nil {
		return nil, nil, xerrors.Errorf("set vesting actor: %w", err)
	}

	// Create pre-allocation actors
	if err := createMultisigAccount(ctx, bs, cst, state, builtin.DefaultGovernorIDAddress, template.DefaultGovernorActor, keyIDs); err != nil {
		return nil, nil, xerrors.Errorf("failed to set up initial governor: %w", err)
	}
	if err := createMultisigAccount(ctx, bs, cst, state, builtin.InvestorIDAddress, template.InvestorAccountActor, keyIDs); err != nil {
		return nil, nil, xerrors.Errorf("failed to set up investor account: %w", err)
	}
	if err := createMultisigAccount(ctx, bs, cst, state, builtin.TeamIDAddress, template.TeamAccountActor, keyIDs); err != nil {
		return nil, nil, xerrors.Errorf("failed to set up team account: %w", err)
	}

	// flush as ForEach works on the HAMT
	if _, err := state.Flush(ctx); err != nil {
		return nil, nil, err
	}

	totalEpkAllocated := big.Zero()
	err = state.ForEach(func(addr address.Address, act *types.Actor) error {
		totalEpkAllocated = big.Add(totalEpkAllocated, act.Balance)
		fmt.Printf("genesis %s balance: %s\n", addr, types.EPK(act.Balance))
		return nil
	})
	if err != nil {
		return nil, nil, xerrors.Errorf("summing account balances in state tree: %w", err)
	}

	totalEpk := big.Mul(big.NewInt(int64(build.EpkBase)), big.NewInt(int64(build.EpkPrecision)))
	template.FoundationAccountActor.Balance = big.Sub(totalEpk, totalEpkAllocated)
	if template.FoundationAccountActor.Balance.Sign() < 0 {
		return nil, nil, xerrors.Errorf("somehow overallocated epk (allocated = %s)", types.EPK(totalEpkAllocated))
	}
	fmt.Printf("genesis foundation balance: %s\n", types.EPK(template.FoundationAccountActor.Balance))

	if err := createMultisigAccount(ctx, bs, cst, state, builtin.FoundationIDAddress, template.FoundationAccountActor, keyIDs); err != nil {
		return nil, nil, xerrors.Errorf("failed to set up foundation account: %w", err)
	}

	return state, keyIDs, nil
}

// returns ID-Address
func createAccountActor(ctx context.Context, cst cbor.IpldStore,
	state *state.StateTree, info genesis.Actor,
	keyIDs map[address.Address]address.Address,
) (address.Address, error) {
	var ainfo genesis.AccountMeta
	if err := json.Unmarshal(info.Meta, &ainfo); err != nil {
		return address.Undef, xerrors.Errorf("unmarshaling account meta: %w", err)
	}

	ida, ok := keyIDs[ainfo.Owner]
	if !ok {
		return address.Undef, fmt.Errorf("no registered ID for account actor: %s", ainfo.Owner)
	}

	// Check if actor already exists
	_, err := state.GetActor(ainfo.Owner)
	if err == nil {
		eid, err := state.LookupID(ainfo.Owner)
		if err != nil {
			return address.Undef, fmt.Errorf("failed to lookup id for account %s", ainfo.Owner)
		}
		if eid != ida {
			return address.Undef, fmt.Errorf("ID address mismatched, exist %s expected %s", eid, ida)
		}
		return ida, nil
	}

	st, err := cst.Put(ctx, &account2.State{Address: ainfo.Owner})
	if err != nil {
		return address.Undef, err
	}
	err = state.SetActor(ida, &types.Actor{
		Code:    builtin2.AccountActorCodeID,
		Balance: info.Balance,
		Head:    st,
	})
	if err != nil {
		return address.Undef, xerrors.Errorf("setting account from actmap: %w", err)
	}
	return ida, nil
}

func createMultisigAccount(ctx context.Context, bs bstore.Blockstore, cst cbor.IpldStore, state *state.StateTree, ida address.Address, info genesis.Actor, keyIDs map[address.Address]address.Address) error {
	if info.Type != genesis.TMultisig {
		return fmt.Errorf("can only call createMultisigAccount with multisig Actor info")
	}
	var ainfo genesis.MultisigMeta
	if err := json.Unmarshal(info.Meta, &ainfo); err != nil {
		return xerrors.Errorf("unmarshaling account meta: %w", err)
	}
	pending, err := adt2.StoreEmptyMap(adt2.WrapStore(ctx, cst), builtin2.DefaultHamtBitwidth)
	if err != nil {
		return xerrors.Errorf("failed to create empty map: %v", err)
	}

	var signers []address.Address

	for _, e := range ainfo.Signers {
		idAddress, ok := keyIDs[e]
		if !ok {
			return fmt.Errorf("no registered key ID for signer: %s", e)
		}

		// Check if actor already exists
		_, err := state.GetActor(e)
		if err == nil {
			signers = append(signers, idAddress)
			continue
		}

		st, err := cst.Put(ctx, &account2.State{Address: e})
		if err != nil {
			return err
		}
		err = state.SetActor(idAddress, &types.Actor{
			Code:    builtin2.AccountActorCodeID,
			Balance: types.NewInt(0),
			Head:    st,
		})
		if err != nil {
			return xerrors.Errorf("setting account from actmap: %w", err)
		}
		signers = append(signers, idAddress)
	}

	st, err := cst.Put(ctx, &multisig2.State{
		Signers:               signers,
		NumApprovalsThreshold: uint64(ainfo.Threshold),
		StartEpoch:            abi.ChainEpoch(ainfo.VestingStart),
		UnlockDuration:        abi.ChainEpoch(ainfo.VestingDuration),
		PendingTxns:           pending,
		InitialBalance:        ainfo.InitialVestingBalance(info.Balance),
	})
	if err != nil {
		return err
	}
	err = state.SetActor(ida, &types.Actor{
		Code:    builtin2.MultisigActorCodeID,
		Balance: info.Balance,
		Head:    st,
	})
	if err != nil {
		return xerrors.Errorf("setting account from actmap: %w", err)
	}
	return nil
}

func VerifyPreSealedData(ctx context.Context, cs *store.ChainStore, stateroot cid.Cid, template genesis.Template, inis *InitDatas, keyIDs map[address.Address]address.Address) (cid.Cid, error) {

	vmopt := vm.VMOpts{
		StateBase:      stateroot,
		Epoch:          0,
		Rand:           &fakeRand{},
		Bstore:         cs.StateBlockstore(),
		Syscalls:       mkFakedSigSyscalls(cs.VMSys()),
		CircSupplyCalc: nil,
		NtwkVersion:    genesisNetworkVersion,
		BaseFee:        types.NewInt(0),
	}
	vm, err := vm.NewVM(ctx, &vmopt)
	if err != nil {
		return cid.Undef, xerrors.Errorf("failed to create NewVM: %w", err)
	}

	// Set initial governor
	{
		_, err = doExecValue(ctx, vm, builtin2.GovernActorAddr, builtin.FoundationIDAddress, types.NewInt(0), builtin2.MethodsGovern.Grant, mustEnc(&govern.GrantOrRevokeParams{
			Governor: builtin.DefaultGovernorIDAddress,
			All:      true,
		}))
		if err != nil {
			return cid.Undef, xerrors.Errorf("failed to set initial governor: %w", err)
		}
	}

	// Set initial expert through power actor
	{
		var ainfo genesis.AccountMeta
		if err := json.Unmarshal(template.DefaultExpertActor.Meta, &ainfo); err != nil {
			return cid.Undef, xerrors.Errorf("unmarshaling expert meta: %w", err)
		}

		expertCreateParams := &expertfund.ApplyForExpertParams{Owner: ainfo.Owner}
		params := mustEnc(expertCreateParams)
		idas, err := ParseIDAddresses(template.FoundationAccountActor, keyIDs)
		if err != nil {
			return cid.Undef, xerrors.Errorf("failed to parse id addresses: %w", err)
		}
		rval, err := doExecValue(ctx, vm, builtin2.ExpertFundActorAddr, idas[0], big.Zero(), builtin2.MethodsExpertFunds.ApplyForExpert, params)
		if err != nil {
			return cid.Undef, xerrors.Errorf("failed to create genesis expert %s: %w", ainfo.Owner, err)
		}
		var ret expertfund.ApplyForExpertReturn
		if err := ret.UnmarshalCBOR(bytes.NewReader(rval)); err != nil {
			return cid.Undef, xerrors.Errorf("unmarshaling ApplyForExpertReturn: %w", err)
		}
		inis.ExpertOwner = ainfo.Owner
		inis.Expert = ret.IDAddress
		fmt.Printf("create genesis expert %s: %s\n", ret.IDAddress, ainfo.Owner)
	}

	// Register genesis file
	pt := template.Miners[0].Sectors[0].ProofType
	if err != nil {
		return cid.Undef, xerrors.Errorf("failed to generate genesis file piece CID: %w", err)
	}
	psize, _ := pt.SectorSize()
	_, err = doExecValue(ctx, vm, inis.Expert, inis.ExpertOwner, big.Zero(),
		builtin2.MethodsExpert.ImportData,
		mustEnc(&expert2.BatchImportDataParams{
			Datas: []expert2.ImportDataParams{{
				RootID:    template.Miners[0].Sectors[0].Deal.PieceCID,
				PieceID:   template.Miners[0].Sectors[0].Deal.PieceCID,
				PieceSize: abi.PaddedPieceSize(psize),
			}},
		}))
	if err != nil {
		return cid.Undef, xerrors.Errorf("failed to import expert data: %w", err)
	}

	// verify pre seal
	for mi, m := range template.Miners {
		if len(m.Sectors) > 1 {
			return cid.Undef, xerrors.Errorf("Should be no more than one sector in genesis miner %d in template: actual %d", mi, len(m.Sectors))
		}
		for si, s := range m.Sectors { // Actually should be only one sector
			if s.Deal.Provider != m.ID {
				return cid.Undef, xerrors.Errorf("Sector %d in miner %d in template had mismatch in provider and miner ID: %s != %s", si, mi, s.Deal.Provider, m.ID)
			}
			if s.ProofType != pt {
				return cid.Undef, xerrors.Errorf("Sector %d in miner %d in template had mismatch in proof type: %s != %s", si, mi, s.ProofType, pt)
			}
		}
	}

	st, err := vm.Flush(ctx)
	if err != nil {
		return cid.Cid{}, xerrors.Errorf("vm flush: %w", err)
	}

	return st, nil
}

func MakeGenesisBlock(ctx context.Context, j journal.Journal, bs bstore.Blockstore, sys vm.SyscallBuilder, template genesis.Template) (*GenesisBootstrap, error) {
	if j == nil {
		j = journal.NilJournal()
	}
	st, keyIDs, err := MakeInitialStateTree(ctx, bs, template)
	if err != nil {
		return nil, xerrors.Errorf("make initial state tree failed: %w", err)
	}

	stateroot, err := st.Flush(ctx)
	if err != nil {
		return nil, xerrors.Errorf("flush state tree failed: %w", err)
	}

	// temp chainstore
	cs := store.NewChainStore(bs, bs, datastore.NewMapDatastore(), sys, j)

	var inis InitDatas
	// Verify PreSealed Data
	stateroot, err = VerifyPreSealedData(ctx, cs, stateroot, template, &inis, keyIDs)
	if err != nil {
		return nil, xerrors.Errorf("failed to verify presealed data: %w", err)
	}

	stateroot, err = SetupStorageMiners(ctx, cs, stateroot, template, inis)
	if err != nil {
		return nil, xerrors.Errorf("setup miners failed: %w", err)
	}

	store := adt2.WrapStore(ctx, cbor.NewCborStore(bs))
	emptyroot, err := adt2.StoreEmptyArray(store, builtin2.DefaultAmtBitwidth)
	if err != nil {
		return nil, xerrors.Errorf("amt build failed: %w", err)
	}

	mm := &types.MsgMeta{
		BlsMessages:   emptyroot,
		SecpkMessages: emptyroot,
	}
	mmb, err := mm.ToStorageBlock()
	if err != nil {
		return nil, xerrors.Errorf("serializing msgmeta failed: %w", err)
	}
	if err := bs.Put(mmb); err != nil {
		return nil, xerrors.Errorf("putting msgmeta block to blockstore: %w", err)
	}

	log.Infof("Empty Genesis root: %s", emptyroot)

	tickBuf := make([]byte, 32)
	_, _ = rand.Read(tickBuf)
	genesisticket := &types.Ticket{
		VRFProof: tickBuf,
	}

	epikGenesisCid, err := cid.Decode(epikGenesisCidString)
	if err != nil {
		return nil, xerrors.Errorf("failed to decode epik genesis block CID: %w", err)
	}

	if !expectedCid().Equals(epikGenesisCid) {
		return nil, xerrors.Errorf("expectedCid != epikGenesisCid")
	}

	gblk, err := getGenesisBlock()
	if err != nil {
		return nil, xerrors.Errorf("failed to construct epik genesis block: %w", err)
	}

	if !epikGenesisCid.Equals(gblk.Cid()) {
		return nil, xerrors.Errorf("epikGenesisCid != gblk.Cid")
	}

	if err := bs.Put(gblk); err != nil {
		return nil, xerrors.Errorf("failed writing epik genesis block to blockstore: %w", err)
	}

	b := &types.BlockHeader{
		Miner:                 builtin2.SystemActorAddr,
		Ticket:                genesisticket,
		Parents:               []cid.Cid{epikGenesisCid},
		Height:                0,
		ParentWeight:          types.NewInt(0),
		ParentStateRoot:       stateroot,
		Messages:              mmb.Cid(),
		ParentMessageReceipts: emptyroot,
		BLSAggregate:          nil,
		BlockSig:              nil,
		Timestamp:             template.Timestamp,
		ElectionProof:         new(types.ElectionProof),
		BeaconEntries: []types.BeaconEntry{
			{
				Round: 0,
				Data:  make([]byte, 32),
			},
		},
		ParentBaseFee: abi.NewTokenAmount(build.InitialBaseFee),
	}

	sb, err := b.ToStorageBlock()
	if err != nil {
		return nil, xerrors.Errorf("serializing block header failed: %w", err)
	}

	if err := bs.Put(sb); err != nil {
		return nil, xerrors.Errorf("putting header to blockstore: %w", err)
	}

	return &GenesisBootstrap{
		Genesis: b,
	}, nil
}
