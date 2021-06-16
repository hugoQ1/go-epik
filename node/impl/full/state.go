package full

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	cid "github.com/ipfs/go-cid"
	"go.uber.org/fx"
	"golang.org/x/xerrors"

	"github.com/EpiK-Protocol/go-epik/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/dline"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/expert"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/expertfund"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/govern"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/knowledge"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/market"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/miner"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/multisig"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/power"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/retrieval"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/reward"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/vesting"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/vote"
	"github.com/EpiK-Protocol/go-epik/chain/beacon"
	"github.com/EpiK-Protocol/go-epik/chain/gen"
	"github.com/EpiK-Protocol/go-epik/chain/state"
	"github.com/EpiK-Protocol/go-epik/chain/stmgr"
	"github.com/EpiK-Protocol/go-epik/chain/store"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/EpiK-Protocol/go-epik/chain/vm"
	"github.com/EpiK-Protocol/go-epik/chain/wallet"
	"github.com/EpiK-Protocol/go-epik/node/modules/dtypes"
	expert2 "github.com/filecoin-project/specs-actors/v2/actors/builtin/expert"
)

type StateModuleAPI interface {
	MsigGetAvailableBalance(ctx context.Context, addr address.Address, tsk types.TipSetKey) (types.BigInt, error)
	MsigGetVested(ctx context.Context, addr address.Address, start types.TipSetKey, end types.TipSetKey) (types.BigInt, error)
	MsigGetPending(ctx context.Context, addr address.Address, tsk types.TipSetKey) ([]*api.MsigTransaction, error)
	StateAccountKey(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error)
	/* StateDealProviderCollateralBounds(ctx context.Context, size abi.PaddedPieceSize, verified bool, tsk types.TipSetKey) (api.DealCollateralBounds, error) */
	StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error)
	StateGetReceipt(context.Context, cid.Cid, types.TipSetKey) (*types.MessageReceipt, error)
	StateListMiners(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error)
	StateLookupID(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error)
	// StateMarketBalance(ctx context.Context, addr address.Address, tsk types.TipSetKey) (api.MarketBalance, error)
	StateMarketStorageDeal(ctx context.Context, dealId abi.DealID, tsk types.TipSetKey) (*api.MarketDeal, error)
	StateMinerInfo(ctx context.Context, actor address.Address, tsk types.TipSetKey) (miner.MinerInfo, error)
	StateMinerProvingDeadline(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*dline.Info, error)
	StateMinerPower(context.Context, address.Address, types.TipSetKey) (*api.MinerPower, error)
	StateNetworkVersion(ctx context.Context, key types.TipSetKey) (network.Version, error)
	StateSearchMsg(ctx context.Context, msg cid.Cid) (*api.MsgLookup, error)
	StateSectorGetInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorOnChainInfo, error)
	// StateVerifiedClientStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error)
	StateWaitMsg(ctx context.Context, msg cid.Cid, confidence uint64) (*api.MsgLookup, error)
}

// StateModule provides a default implementation of StateModuleAPI.
// It can be swapped out with another implementation through Dependency
// Injection (for example with a thin RPC client).
type StateModule struct {
	fx.In

	StateManager *stmgr.StateManager
	Chain        *store.ChainStore
}

var _ StateModuleAPI = (*StateModule)(nil)

type StateAPI struct {
	fx.In

	// TODO: the wallet here is only needed because we have the MinerCreateBlock
	// API attached to the state API. It probably should live somewhere better
	Wallet    api.WalletAPI
	DefWallet wallet.Default

	StateModuleAPI

	ProofVerifier ffiwrapper.Verifier
	StateManager  *stmgr.StateManager
	Chain         *store.ChainStore
	Beacon        beacon.Schedule
}

func (a *StateAPI) StateNetworkName(ctx context.Context) (dtypes.NetworkName, error) {
	return stmgr.GetNetworkName(ctx, a.StateManager, a.Chain.GetHeaviestTipSet().ParentState())
}

func (a *StateAPI) StateMinerSectors(ctx context.Context, addr address.Address, sectorNos *bitfield.BitField, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, addr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return mas.LoadSectors(sectorNos)
}

func (a *StateAPI) StateMinerActiveSectors(ctx context.Context, maddr address.Address, tsk types.TipSetKey) ([]*miner.SectorOnChainInfo, error) { // TODO: only used in cli
	act, err := a.StateManager.LoadActorTsk(ctx, maddr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	activeSectors, err := miner.AllPartSectors(mas, miner.Partition.ActiveSectors)
	if err != nil {
		return nil, xerrors.Errorf("merge partition active sets: %w", err)
	}

	return mas.LoadSectors(&activeSectors)
}

func (m *StateModule) StateMinerInfo(ctx context.Context, actor address.Address, tsk types.TipSetKey) (miner.MinerInfo, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return miner.MinerInfo{}, xerrors.Errorf("failed to load tipset: %w", err)
	}

	act, err := m.StateManager.LoadActor(ctx, actor, ts)
	if err != nil {
		return miner.MinerInfo{}, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(m.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return miner.MinerInfo{}, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	info, err := mas.Info()
	if err != nil {
		return miner.MinerInfo{}, err
	}

	// coinbase
	cbact, err := m.StateManager.LoadActor(ctx, builtin.VestingActorAddr, ts)
	if err != nil {
		return miner.MinerInfo{}, xerrors.Errorf("failed to load coinbase actor: %w", err)
	}
	cbs, err := vesting.Load(m.StateManager.ChainStore().ActorStore(ctx), cbact)
	if err != nil {
		return miner.MinerInfo{}, xerrors.Errorf("failed to load coinbase actor state: %w", err)
	}
	accMined, err := cbs.TotalMined(actor)
	if err != nil {
		return miner.MinerInfo{}, err
	}
	info.TotalMined = accMined

	return info, nil
}

func (a *StateAPI) StateMinerDeadlines(ctx context.Context, m address.Address, tsk types.TipSetKey) ([]api.Deadline, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, m, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	deadlines, err := mas.NumDeadlines()
	if err != nil {
		return nil, xerrors.Errorf("getting deadline count: %w", err)
	}

	out := make([]api.Deadline, deadlines)
	if err := mas.ForEachDeadline(func(i uint64, dl miner.Deadline) error {
		ps, err := dl.PartitionsPoSted()
		if err != nil {
			return err
		}

		l, err := dl.DisputableProofCount()
		if err != nil {
			return err
		}

		out[i] = api.Deadline{
			PostSubmissions:      ps,
			DisputableProofCount: l,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *StateAPI) StateMinerPartitions(ctx context.Context, m address.Address, dlIdx uint64, tsk types.TipSetKey) ([]api.Partition, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, m, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	dl, err := mas.LoadDeadline(dlIdx)
	if err != nil {
		return nil, xerrors.Errorf("failed to load the deadline: %w", err)
	}

	var out []api.Partition
	err = dl.ForEachPartition(func(_ uint64, part miner.Partition) error {
		allSectors, err := part.AllSectors()
		if err != nil {
			return xerrors.Errorf("getting AllSectors: %w", err)
		}

		faultySectors, err := part.FaultySectors()
		if err != nil {
			return xerrors.Errorf("getting FaultySectors: %w", err)
		}

		recoveringSectors, err := part.RecoveringSectors()
		if err != nil {
			return xerrors.Errorf("getting RecoveringSectors: %w", err)
		}

		liveSectors, err := part.LiveSectors()
		if err != nil {
			return xerrors.Errorf("getting LiveSectors: %w", err)
		}

		activeSectors, err := part.ActiveSectors()
		if err != nil {
			return xerrors.Errorf("getting ActiveSectors: %w", err)
		}

		out = append(out, api.Partition{
			AllSectors:        allSectors,
			FaultySectors:     faultySectors,
			RecoveringSectors: recoveringSectors,
			LiveSectors:       liveSectors,
			ActiveSectors:     activeSectors,
		})
		return nil
	})

	return out, err
}

func (m *StateModule) StateMinerProvingDeadline(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*dline.Info, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := m.StateManager.LoadActor(ctx, addr, ts)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(m.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	di, err := mas.DeadlineInfo(ts.Height())
	if err != nil {
		return nil, xerrors.Errorf("failed to get deadline info: %w", err)
	}

	return di.NextNotElapsed(), nil
}

func (a *StateAPI) StateMinerFaults(ctx context.Context, addr address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, addr, tsk)
	if err != nil {
		return bitfield.BitField{}, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return bitfield.BitField{}, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return miner.AllPartSectors(mas, miner.Partition.FaultySectors)
}

func (a *StateAPI) StateMinerActives(ctx context.Context, addr address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, addr, tsk)
	if err != nil {
		if err == types.ErrActorNotFound {
			return bitfield.BitField{}, nil
		}
		return bitfield.BitField{}, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return bitfield.BitField{}, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return miner.AllPartSectors(mas, miner.Partition.ActiveSectors)
}

func (a *StateAPI) StateAllMinerFaults(ctx context.Context, lookback abi.ChainEpoch, endTsk types.TipSetKey) ([]*api.Fault, error) {
	return nil, xerrors.Errorf("fixme")

	/*endTs, err := a.Chain.GetTipSetFromKey(endTsk)
	if err != nil {
		return nil, xerrors.Errorf("loading end tipset %s: %w", endTsk, err)
	}

	cutoff := endTs.Height() - lookback
	miners, err := stmgr.ListMinerActors(ctx, a.StateManager, endTs)

	if err != nil {
		return nil, xerrors.Errorf("loading miners: %w", err)
	}

	var allFaults []*api.Fault

	for _, m := range miners {
		var mas miner.State
		_, err := a.StateManager.LoadActorState(ctx, m, &mas, endTs)
		if err != nil {
			return nil, xerrors.Errorf("failed to load miner actor state %s: %w", m, err)
		}

		err = mas.ForEachFaultEpoch(a.Chain.ActorStore(ctx), func(faultStart abi.ChainEpoch, faults abi.BitField) error {
			if faultStart >= cutoff {
				allFaults = append(allFaults, &api.Fault{
					Miner: m,
					Epoch: faultStart,
				})
				return nil
			}
			return nil
		})

		if err != nil {
			return nil, xerrors.Errorf("failure when iterating over miner states: %w", err)
		}
	}

	return allFaults, nil*/
}

func (a *StateAPI) StateMinerRecoveries(ctx context.Context, addr address.Address, tsk types.TipSetKey) (bitfield.BitField, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, addr, tsk)
	if err != nil {
		return bitfield.BitField{}, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return bitfield.BitField{}, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return miner.AllPartSectors(mas, miner.Partition.RecoveringSectors)
}

func (m *StateModule) StateMinerPower(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.MinerPower, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	mp, net, hmp, err := stmgr.GetPower(ctx, m.StateManager, ts, addr)
	if err != nil {
		return nil, err
	}

	return &api.MinerPower{
		MinerPower:  mp,
		TotalPower:  net,
		HasMinPower: hmp,
	}, nil
}

func (a *StateAPI) StateCall(ctx context.Context, msg *types.Message, tsk types.TipSetKey) (res *api.InvocResult, err error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	for {
		res, err = a.StateManager.Call(ctx, msg, ts)
		if err != stmgr.ErrExpensiveFork {
			break
		}
		ts, err = a.Chain.GetTipSetFromKey(ts.Parents())
		if err != nil {
			return nil, xerrors.Errorf("getting parent tipset: %w", err)
		}
	}
	return res, err
}

func (a *StateAPI) StateReplay(ctx context.Context, tsk types.TipSetKey, mc cid.Cid) (*api.InvocResult, error) {
	msgToReplay := mc
	var ts *types.TipSet
	var err error
	if tsk == types.EmptyTSK {
		mlkp, err := a.StateSearchMsg(ctx, mc)
		if err != nil {
			return nil, xerrors.Errorf("searching for msg %s: %w", mc, err)
		}
		if mlkp == nil {
			return nil, xerrors.Errorf("didn't find msg %s", mc)
		}

		msgToReplay = mlkp.Message

		executionTs, err := a.Chain.GetTipSetFromKey(mlkp.TipSet)
		if err != nil {
			return nil, xerrors.Errorf("loading tipset %s: %w", mlkp.TipSet, err)
		}

		ts, err = a.Chain.LoadTipSet(executionTs.Parents())
		if err != nil {
			return nil, xerrors.Errorf("loading parent tipset %s: %w", mlkp.TipSet, err)
		}
	} else {
		ts, err = a.Chain.LoadTipSet(tsk)
		if err != nil {
			return nil, xerrors.Errorf("loading specified tipset %s: %w", tsk, err)
		}
	}

	m, r, err := a.StateManager.Replay(ctx, ts, msgToReplay)
	if err != nil {
		return nil, err
	}

	var errstr string
	if r.ActorErr != nil {
		errstr = r.ActorErr.Error()
	}

	return &api.InvocResult{
		MsgCid:         msgToReplay,
		Msg:            m,
		MsgRct:         &r.MessageReceipt,
		GasCost:        stmgr.MakeMsgGasCost(m, r),
		ExecutionTrace: r.ExecutionTrace,
		Error:          errstr,
		Duration:       r.Duration,
	}, nil
}

func (a *StateAPI) StateBlockReward(ctx context.Context, bid cid.Cid, tsk types.TipSetKey) (*api.BlockReward, error) {
	header, err := a.Chain.GetBlock(bid)
	if err != nil {
		return nil, err
	}

	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading specified tipset %s: %w", tsk, err)
	}
	bts, err := a.Chain.GetTipsetByHeight(ctx, header.Height, ts, false)
	if err != nil {
		return nil, xerrors.Errorf("get tipset by height %d: %w", header.Height, err)
	}
	if bts.Height() != header.Height {
		return nil, xerrors.Errorf("null round block height: %d", header.Height)
	}

	found := 0
	nth := 0
	for i, id := range bts.Cids() {
		if id == bid {
			found++
			nth = i
		}
	}
	switch found {
	case 1:
		return a.StateManager.ReplayBlock(ctx, nth, bts)
	case 0:
		return nil, xerrors.New("block not found")
	default:
		return nil, xerrors.Errorf("more than 1 blocks found with the same cid:%s", bid)
	}
}

func (m *StateModule) StateGetActor(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*types.Actor, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return m.StateManager.LoadActor(ctx, actor, ts)
}

func (m *StateModule) StateLookupID(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return address.Undef, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	return m.StateManager.LookupID(ctx, addr, ts)
}

func (m *StateModule) StateAccountKey(ctx context.Context, addr address.Address, tsk types.TipSetKey) (address.Address, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return address.Undef, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	return m.StateManager.ResolveToKeyAddress(ctx, addr, ts)
}

func (a *StateAPI) StateReadState(ctx context.Context, actor address.Address, tsk types.TipSetKey) (*api.ActorState, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	act, err := a.StateManager.LoadActor(ctx, actor, ts)
	if err != nil {
		return nil, xerrors.Errorf("getting actor: %w", err)
	}

	blk, err := a.Chain.StateBlockstore().Get(act.Head)
	if err != nil {
		return nil, xerrors.Errorf("getting actor head: %w", err)
	}

	oif, err := vm.DumpActorState(act, blk.RawData())
	if err != nil {
		return nil, xerrors.Errorf("dumping actor state (a:%s): %w", actor, err)
	}

	return &api.ActorState{
		Balance: act.Balance,
		Code:    act.Code,
		State:   oif,
	}, nil
}

func (a *StateAPI) StateDecodeParams(ctx context.Context, toAddr address.Address, method abi.MethodNum, params []byte, tsk types.TipSetKey) (interface{}, error) {
	act, err := a.StateGetActor(ctx, toAddr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("getting actor: %w", err)
	}

	paramType, err := stmgr.GetParamType(act.Code, method)
	if err != nil {
		return nil, xerrors.Errorf("getting params type: %w", err)
	}

	if err = paramType.UnmarshalCBOR(bytes.NewReader(params)); err != nil {
		return nil, err
	}

	return paramType, nil
}

// This is on StateAPI because miner.Miner requires this, and MinerAPI requires miner.Miner
func (a *StateAPI) MinerGetBaseInfo(ctx context.Context, maddr address.Address, epoch abi.ChainEpoch, tsk types.TipSetKey) (*api.MiningBaseInfo, error) {
	// XXX: Gets the state by computing the tipset state, instead of looking at the parent.
	return stmgr.MinerGetBaseInfo(ctx, a.StateManager, a.Beacon, tsk, epoch, maddr, a.ProofVerifier)
}

func (a *StateAPI) MinerCreateBlock(ctx context.Context, bt *api.BlockTemplate) (*types.BlockMsg, error) {
	fblk, err := gen.MinerCreateBlock(ctx, a.StateManager, a.Wallet, bt)
	if err != nil {
		return nil, err
	}

	var out types.BlockMsg
	out.Header = fblk.Header
	for _, msg := range fblk.BlsMessages {
		out.BlsMessages = append(out.BlsMessages, msg.Cid())
	}
	for _, msg := range fblk.SecpkMessages {
		out.SecpkMessages = append(out.SecpkMessages, msg.Cid())
	}

	return &out, nil
}

func (m *StateModule) StateWaitMsg(ctx context.Context, msg cid.Cid, confidence uint64) (*api.MsgLookup, error) {
	return stateWaitMsgLimited(ctx, m.StateManager, m.Chain, msg, confidence, stmgr.LookbackNoLimit)
}
func (a *StateAPI) StateWaitMsgLimited(ctx context.Context, msg cid.Cid, confidence uint64, lookbackLimit abi.ChainEpoch) (*api.MsgLookup, error) {
	return stateWaitMsgLimited(ctx, a.StateManager, a.Chain, msg, confidence, lookbackLimit)
}
func stateWaitMsgLimited(ctx context.Context, smgr *stmgr.StateManager, cstore *store.ChainStore, msg cid.Cid, confidence uint64, lookbackLimit abi.ChainEpoch) (*api.MsgLookup, error) {
	ts, recpt, found, err := smgr.WaitForMessage(ctx, msg, confidence, lookbackLimit)
	if err != nil {
		return nil, err
	}

	var returndec interface{}
	if recpt.ExitCode == 0 && len(recpt.Return) > 0 {
		cmsg, err := cstore.GetCMessage(msg)
		if err != nil {
			return nil, xerrors.Errorf("failed to load message after successful receipt search: %w", err)
		}

		vmsg := cmsg.VMMessage()

		t, err := stmgr.GetReturnType(ctx, smgr, vmsg.To, vmsg.Method, ts)
		if err != nil {
			return nil, xerrors.Errorf("failed to get return type: %w", err)
		}

		if err := t.UnmarshalCBOR(bytes.NewReader(recpt.Return)); err != nil {
			return nil, err
		}

		returndec = t
	}

	return &api.MsgLookup{
		Message:   found,
		Receipt:   *recpt,
		ReturnDec: returndec,
		TipSet:    ts.Key(),
		Height:    ts.Height(),
	}, nil
}

func (m *StateModule) StateSearchMsg(ctx context.Context, msg cid.Cid) (*api.MsgLookup, error) {
	return stateSearchMsgLimited(ctx, m.StateManager, msg, stmgr.LookbackNoLimit)
}
func (a *StateAPI) StateSearchMsgLimited(ctx context.Context, msg cid.Cid, lookbackLimit abi.ChainEpoch) (*api.MsgLookup, error) {
	return stateSearchMsgLimited(ctx, a.StateManager, msg, lookbackLimit)
}
func stateSearchMsgLimited(ctx context.Context, smgr *stmgr.StateManager, msg cid.Cid, lookbackLimit abi.ChainEpoch) (*api.MsgLookup, error) {
	ts, recpt, found, err := smgr.SearchForMessage(ctx, msg, lookbackLimit)
	if err != nil {
		return nil, err
	}

	if ts != nil {
		return &api.MsgLookup{
			Message: found,
			Receipt: *recpt,
			TipSet:  ts.Key(),
			Height:  ts.Height(),
		}, nil
	}
	return nil, nil
}

func (m *StateModule) StateGetReceipt(ctx context.Context, msg cid.Cid, tsk types.TipSetKey) (*types.MessageReceipt, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return m.StateManager.GetReceipt(ctx, msg, ts)
}

func (m *StateModule) StateListMiners(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return stmgr.ListMinerActors(ctx, m.StateManager, ts)
}

func (a *StateAPI) StateListActors(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return a.StateManager.ListAllActors(ctx, ts)
}

// func (m *StateModule) StateMarketBalance(ctx context.Context, addr address.Address, tsk types.TipSetKey) (api.MarketBalance, error) {
// 	ts, err := m.Chain.GetTipSetFromKey(tsk)
// 	if err != nil {
// 		return api.MarketBalance{}, xerrors.Errorf("loading tipset %s: %w", tsk, err)
// 	}
// 	return m.StateManager.MarketBalance(ctx, addr, ts)
// }

// func (a *StateAPI) StateMarketParticipants(ctx context.Context, tsk types.TipSetKey) (map[string]api.MarketBalance, error) {
// 	out := map[string]api.MarketBalance{}

// 	ts, err := a.Chain.GetTipSetFromKey(tsk)
// 	if err != nil {
// 		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
// 	}

// 	state, err := a.StateManager.GetMarketState(ctx, ts)
// 	if err != nil {
// 		return nil, err
// 	}
// 	escrow, err := state.EscrowTable()
// 	if err != nil {
// 		return nil, err
// 	}
// 	locked, err := state.LockedTable()
// 	if err != nil {
// 		return nil, err
// 	}

// 	err = escrow.ForEach(func(a address.Address, es abi.TokenAmount) error {

// 		lk, err := locked.Get(a)
// 		if err != nil {
// 			return err
// 		}

// 		out[a.String()] = api.MarketBalance{
// 			Escrow: es,
// 			Locked: lk,
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
// 	return out, nil
// }

func (a *StateAPI) StateMarketDeals(ctx context.Context, tsk types.TipSetKey) (map[string]api.MarketDeal, error) {
	out := map[string]api.MarketDeal{}

	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	state, err := a.StateManager.GetMarketState(ctx, ts)
	if err != nil {
		return nil, err
	}

	da, err := state.Proposals()
	if err != nil {
		return nil, err
	}

	sa, err := state.States()
	if err != nil {
		return nil, err
	}

	if err := da.ForEach(func(dealID abi.DealID, d market.DealProposal) error {
		s, found, err := sa.Get(dealID)
		if err != nil {
			return xerrors.Errorf("failed to get state for deal in proposals array: %w", err)
		} else if !found {
			s = market.EmptyDealState()
		}
		out[strconv.FormatInt(int64(dealID), 10)] = api.MarketDeal{
			Proposal: d,
			State:    *s,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *StateModule) StateMarketStorageDeal(ctx context.Context, dealId abi.DealID, tsk types.TipSetKey) (*api.MarketDeal, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return stmgr.GetStorageDeal(ctx, m.StateManager, dealId, ts)
}

func (m *StateAPI) StateMarketRemainingQuota(ctx context.Context, pieceCid cid.Cid, tsk types.TipSetKey) (int64, error) {
	act, err := m.StateManager.LoadActorTsk(ctx, market.Address, tsk)
	if err != nil {
		return 0, xerrors.Errorf("failed to load market actor: %w", err)
	}

	mState, err := market.Load(m.Chain.ActorStore(ctx), act)
	if err != nil {
		return 0, xerrors.Errorf("failed to load market actor state: %w", err)
	}

	qs, err := mState.Quotas()
	if err != nil {
		return 0, xerrors.Errorf("failed to get quotas: %w", err)
	}
	return qs.RemainingQuota(pieceCid)
}

func (a *StateAPI) StateChangedActors(ctx context.Context, old cid.Cid, new cid.Cid) (map[string]types.Actor, error) {
	store := a.Chain.ActorStore(ctx)

	oldTree, err := state.LoadStateTree(store, old)
	if err != nil {
		return nil, xerrors.Errorf("failed to load old state tree: %w", err)
	}

	newTree, err := state.LoadStateTree(store, new)
	if err != nil {
		return nil, xerrors.Errorf("failed to load new state tree: %w", err)
	}

	return state.Diff(oldTree, newTree)
}

func (a *StateAPI) StateMinerSectorCount(ctx context.Context, addr address.Address, tsk types.TipSetKey) (api.MinerSectors, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, addr, tsk)
	if err != nil {
		return api.MinerSectors{}, err
	}
	mas, err := miner.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return api.MinerSectors{}, err
	}
	var activeCount, liveCount, faultyCount uint64
	if err := mas.ForEachDeadline(func(_ uint64, dl miner.Deadline) error {
		return dl.ForEachPartition(func(_ uint64, part miner.Partition) error {
			if active, err := part.ActiveSectors(); err != nil {
				return err
			} else if count, err := active.Count(); err != nil {
				return err
			} else {
				activeCount += count
			}
			if live, err := part.LiveSectors(); err != nil {
				return err
			} else if count, err := live.Count(); err != nil {
				return err
			} else {
				liveCount += count
			}
			if faulty, err := part.FaultySectors(); err != nil {
				return err
			} else if count, err := faulty.Count(); err != nil {
				return err
			} else {
				faultyCount += count
			}
			return nil
		})
	}); err != nil {
		return api.MinerSectors{}, err
	}
	return api.MinerSectors{Live: liveCount, Active: activeCount, Faulty: faultyCount}, nil
}

func (a *StateAPI) StateSectorPreCommitInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (miner.SectorPreCommitOnChainInfo, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return miner.SectorPreCommitOnChainInfo{}, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	pci, err := stmgr.PreCommitInfo(ctx, a.StateManager, maddr, n, ts)
	if err != nil {
		return miner.SectorPreCommitOnChainInfo{}, err
	} else if pci == nil {
		return miner.SectorPreCommitOnChainInfo{}, xerrors.Errorf("precommit info is not exists")
	}
	return *pci, err
}

func (m *StateModule) StateSectorGetInfo(ctx context.Context, maddr address.Address, n abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorOnChainInfo, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return stmgr.MinerSectorInfo(ctx, m.StateManager, maddr, n, ts)
}

func (a *StateAPI) StateSectorExpiration(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorExpiration, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, maddr, tsk)
	if err != nil {
		return nil, err
	}
	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, err
	}
	return mas.GetSectorExpiration(sectorNumber)
}

func (a *StateAPI) StateSectorPartition(ctx context.Context, maddr address.Address, sectorNumber abi.SectorNumber, tsk types.TipSetKey) (*miner.SectorLocation, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, maddr, tsk)
	if err != nil {
		return nil, err
	}
	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, err
	}
	return mas.FindSector(sectorNumber)
}

func (a *StateAPI) StateListMessages(ctx context.Context, match *api.MessageMatch, tsk types.TipSetKey, toheight abi.ChainEpoch) ([]cid.Cid, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	if ts == nil {
		ts = a.Chain.GetHeaviestTipSet()
	}

	if match.To == address.Undef && match.From == address.Undef {
		return nil, xerrors.Errorf("must specify at least To or From in message filter")
	}

	matchFunc := func(msg *types.Message) bool {
		if match.From != address.Undef && match.From != msg.From {
			return false
		}

		if match.To != address.Undef && match.To != msg.To {
			return false
		}

		return true
	}

	var out []cid.Cid
	for ts.Height() >= toheight {
		msgs, err := a.Chain.MessagesForTipset(ts)
		if err != nil {
			return nil, xerrors.Errorf("failed to get messages for tipset (%s): %w", ts.Key(), err)
		}

		for _, msg := range msgs {
			if matchFunc(msg.VMMessage()) {
				out = append(out, msg.Cid())
			}
		}

		if ts.Height() == 0 {
			break
		}

		next, err := a.Chain.LoadTipSet(ts.Parents())
		if err != nil {
			return nil, xerrors.Errorf("loading next tipset: %w", err)
		}

		ts = next
	}

	return out, nil
}

func (a *StateAPI) StateCompute(ctx context.Context, height abi.ChainEpoch, msgs []*types.Message, tsk types.TipSetKey) (*api.ComputeStateOutput, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	st, t, err := stmgr.ComputeState(ctx, a.StateManager, height, msgs, ts)
	if err != nil {
		return nil, err
	}

	return &api.ComputeStateOutput{
		Root:  st,
		Trace: t,
	}, nil
}

func (m *StateModule) MsigGetAvailableBalance(ctx context.Context, addr address.Address, tsk types.TipSetKey) (types.BigInt, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := m.StateManager.LoadActor(ctx, addr, ts)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to load multisig actor: %w", err)
	}
	msas, err := multisig.Load(m.Chain.ActorStore(ctx), act)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to load multisig actor state: %w", err)
	}
	locked, err := msas.LockedBalance(ts.Height())
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to compute locked multisig balance: %w", err)
	}
	return types.BigSub(act.Balance, locked), nil
}

func (a *StateAPI) MsigGetVestingSchedule(ctx context.Context, addr address.Address, tsk types.TipSetKey) (api.MsigVesting, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return api.EmptyVesting, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := a.StateManager.LoadActor(ctx, addr, ts)
	if err != nil {
		return api.EmptyVesting, xerrors.Errorf("failed to load multisig actor: %w", err)
	}

	msas, err := multisig.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return api.EmptyVesting, xerrors.Errorf("failed to load multisig actor state: %w", err)
	}

	ib, err := msas.InitialBalance()
	if err != nil {
		return api.EmptyVesting, xerrors.Errorf("failed to load multisig initial balance: %w", err)
	}

	se, err := msas.StartEpoch()
	if err != nil {
		return api.EmptyVesting, xerrors.Errorf("failed to load multisig start epoch: %w", err)
	}

	ud, err := msas.UnlockDuration()
	if err != nil {
		return api.EmptyVesting, xerrors.Errorf("failed to load multisig unlock duration: %w", err)
	}

	return api.MsigVesting{
		InitialBalance: ib,
		StartEpoch:     se,
		UnlockDuration: ud,
	}, nil
}

func (m *StateModule) MsigGetVested(ctx context.Context, addr address.Address, start types.TipSetKey, end types.TipSetKey) (types.BigInt, error) {
	startTs, err := m.Chain.GetTipSetFromKey(start)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading start tipset %s: %w", start, err)
	}

	endTs, err := m.Chain.GetTipSetFromKey(end)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading end tipset %s: %w", end, err)
	}

	if startTs.Height() > endTs.Height() {
		return types.EmptyInt, xerrors.Errorf("start tipset %d is after end tipset %d", startTs.Height(), endTs.Height())
	} else if startTs.Height() == endTs.Height() {
		return big.Zero(), nil
	}

	act, err := m.StateManager.LoadActor(ctx, addr, endTs)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to load multisig actor at end epoch: %w", err)
	}

	msas, err := multisig.Load(m.Chain.ActorStore(ctx), act)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to load multisig actor state: %w", err)
	}

	startLk, err := msas.LockedBalance(startTs.Height())
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to compute locked balance at start height: %w", err)
	}

	endLk, err := msas.LockedBalance(endTs.Height())
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to compute locked balance at end height: %w", err)
	}

	return types.BigSub(startLk, endLk), nil
}

func (m *StateModule) MsigGetPending(ctx context.Context, addr address.Address, tsk types.TipSetKey) ([]*api.MsigTransaction, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := m.StateManager.LoadActor(ctx, addr, ts)
	if err != nil {
		return nil, xerrors.Errorf("failed to load multisig actor: %w", err)
	}
	msas, err := multisig.Load(m.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load multisig actor state: %w", err)
	}

	var out = []*api.MsigTransaction{}
	if err := msas.ForEachPendingTxn(func(id int64, txn multisig.Transaction) error {
		out = append(out, &api.MsigTransaction{
			ID:     id,
			To:     txn.To,
			Value:  txn.Value,
			Method: txn.Method,
			Params: txn.Params,

			Approved: txn.Approved,
		})
		return nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}

/* var initialPledgeNum = types.NewInt(110)
var initialPledgeDen = types.NewInt(100)

func (a *StateAPI) StateMinerPreCommitDepositForPower(ctx context.Context, maddr address.Address, pci miner.SectorPreCommitInfo, tsk types.TipSetKey) (types.BigInt, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	state, err := a.StateManager.ParentState(ts)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading state %s: %w", tsk, err)
	}

	ssize, err := pci.SealProof.SectorSize()
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to get resolve size: %w", err)
	}

	store := a.Chain.ActorStore(ctx)

	var sectorWeight abi.StoragePower
	if act, err := state.GetActor(market.Address); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading market actor %s: %w", maddr, err)
	} else if s, err := market.Load(store, act); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading market actor state %s: %w", maddr, err)
	} else if w, vw, err := s.VerifyDealsForActivation(maddr, pci.DealIDs, ts.Height(), pci.Expiration); err != nil {
		return types.EmptyInt, xerrors.Errorf("verifying deals for activation: %w", err)
	} else {
		// NB: not exactly accurate, but should always lead us to *over* estimate, not under
		duration := pci.Expiration - ts.Height()
		sectorWeight = builtin.QAPowerForWeight(ssize, duration, w, vw)
	}

	var powerSmoothed builtin.FilterEstimate
	if act, err := state.GetActor(power.Address); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading power actor: %w", err)
	} else if s, err := power.Load(store, act); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading power actor state: %w", err)
	} else if p, err := s.TotalPowerSmoothed(); err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to determine total power: %w", err)
	} else {
		powerSmoothed = p
	}

	rewardActor, err := state.GetActor(reward.Address)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading miner actor: %w", err)
	}

	rewardState, err := reward.Load(store, rewardActor)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading reward actor state: %w", err)
	}

	deposit, err := rewardState.PreCommitDepositForPower(powerSmoothed, sectorWeight)
	if err != nil {
		return big.Zero(), xerrors.Errorf("calculating precommit deposit: %w", err)
	}

	return types.BigDiv(types.BigMul(deposit, initialPledgeNum), initialPledgeDen), nil
}

func (a *StateAPI) StateMinerInitialPledgeCollateral(ctx context.Context, maddr address.Address, pci miner.SectorPreCommitInfo, tsk types.TipSetKey) (types.BigInt, error) {
	// TODO: this repeats a lot of the previous function. Fix that.
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	state, err := a.StateManager.ParentState(ts)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading state %s: %w", tsk, err)
	}

	ssize, err := pci.SealProof.SectorSize()
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to get resolve size: %w", err)
	}

	store := a.Chain.ActorStore(ctx)

	var sectorWeight abi.StoragePower
	if act, err := state.GetActor(market.Address); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading market actor: %w", err)
	} else if s, err := market.Load(store, act); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading market actor state: %w", err)
	} else if w, vw, err := s.VerifyDealsForActivation(maddr, pci.DealIDs, ts.Height(), pci.Expiration); err != nil {
		return types.EmptyInt, xerrors.Errorf("verifying deals for activation: %w", err)
	} else {
		// NB: not exactly accurate, but should always lead us to *over* estimate, not under
		duration := pci.Expiration - ts.Height()
		sectorWeight = builtin.QAPowerForWeight(ssize, duration, w, vw)
	}

	var (
		powerSmoothed    builtin.FilterEstimate
		pledgeCollateral abi.TokenAmount
	)
	if act, err := state.GetActor(power.Address); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading power actor: %w", err)
	} else if s, err := power.Load(store, act); err != nil {
		return types.EmptyInt, xerrors.Errorf("loading power actor state: %w", err)
	} else if p, err := s.TotalPowerSmoothed(); err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to determine total power: %w", err)
	} else if c, err := s.TotalLocked(); err != nil {
		return types.EmptyInt, xerrors.Errorf("failed to determine pledge collateral: %w", err)
	} else {
		powerSmoothed = p
		pledgeCollateral = c
	}

	rewardActor, err := state.GetActor(reward.Address)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading reward actor: %w", err)
	}

	rewardState, err := reward.Load(store, rewardActor)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading reward actor state: %w", err)
	}

	circSupply, err := a.StateVMCirculatingSupplyInternal(ctx, ts.Key())
	if err != nil {
		return big.Zero(), xerrors.Errorf("getting circulating supply: %w", err)
	}

	initialPledge, err := rewardState.InitialPledgeForPower(
		sectorWeight,
		pledgeCollateral,
		&powerSmoothed,
		circSupply.EpkCirculating,
	)
	if err != nil {
		return big.Zero(), xerrors.Errorf("calculating initial pledge: %w", err)
	}

	return types.BigDiv(types.BigMul(initialPledge, initialPledgeNum), initialPledgeDen), nil
} */

func (a *StateAPI) StateCoinbase(ctx context.Context, coinbase address.Address, tsk types.TipSetKey) (*vesting.CoinbaseInfo, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := a.StateManager.LoadActor(ctx, builtin.VestingActorAddr, ts)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	vas, err := vesting.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return vas.Coinbase(coinbase, ts.Height())
}

func (a *StateAPI) StateMinerFunds(ctx context.Context, maddr address.Address, tsk types.TipSetKey) (miner.Funds, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return miner.Funds{}, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := a.StateManager.LoadActor(ctx, maddr, ts)
	if err != nil {
		return miner.Funds{}, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return miner.Funds{}, xerrors.Errorf("failed to load miner actor state: %w", err)
	}
	return mas.Funds()
}

func (a *StateAPI) StateMinerSectorAllocated(ctx context.Context, maddr address.Address, s abi.SectorNumber, tsk types.TipSetKey) (bool, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return false, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	act, err := a.StateManager.LoadActor(ctx, maddr, ts)
	if err != nil {
		return false, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	mas, err := miner.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return false, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return mas.IsAllocated(s)
}

/* // StateVerifiedClientStatus returns the data cap for the given address.
// Returns zero if there is no entry in the data cap table for the
// address.
func (a *StateAPI) StateVerifierStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error) {
	act, err := a.StateGetActor(ctx, verifreg.Address, tsk)
	if err != nil {
		return nil, err
	}

	aid, err := a.StateLookupID(ctx, addr, tsk)
	if err != nil {
		log.Warnf("lookup failure %v", err)
		return nil, err
	}

	vrs, err := verifreg.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load verified registry state: %w", err)
	}

	verified, dcap, err := vrs.VerifierDataCap(aid)
	if err != nil {
		return nil, xerrors.Errorf("looking up verifier: %w", err)
	}
	if !verified {
		return nil, nil
	}

	return &dcap, nil
}

// StateVerifiedClientStatus returns the data cap for the given address.
// Returns zero if there is no entry in the data cap table for the
// address.
func (m *StateModule) StateVerifiedClientStatus(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*abi.StoragePower, error) {
	act, err := m.StateGetActor(ctx, verifreg.Address, tsk)
	if err != nil {
		return nil, err
	}

	aid, err := m.StateLookupID(ctx, addr, tsk)
	if err != nil {
		log.Warnf("lookup failure %v", err)
		return nil, err
	}

	vrs, err := verifreg.Load(m.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load verified registry state: %w", err)
	}

	verified, dcap, err := vrs.VerifiedClientDataCap(aid)
	if err != nil {
		return nil, xerrors.Errorf("looking up verified client: %w", err)
	}
	if !verified {
		return nil, nil
	}

	return &dcap, nil
}

func (a *StateAPI) StateVerifiedRegistryRootKey(ctx context.Context, tsk types.TipSetKey) (address.Address, error) {
	vact, err := a.StateGetActor(ctx, verifreg.Address, tsk)
	if err != nil {
		return address.Undef, err
	}

	vst, err := verifreg.Load(a.StateManager.ChainStore().ActorStore(ctx), vact)
	if err != nil {
		return address.Undef, err
	}

	return vst.RootKey()
}

var dealProviderCollateralNum = types.NewInt(110)
var dealProviderCollateralDen = types.NewInt(100)

// StateDealProviderCollateralBounds returns the min and max collateral a storage provider
// can issue. It takes the deal size and verified status as parameters.
func (m *StateModule) StateDealProviderCollateralBounds(ctx context.Context, size abi.PaddedPieceSize, verified bool, tsk types.TipSetKey) (api.DealCollateralBounds, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	pact, err := m.StateGetActor(ctx, power.Address, tsk)
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("failed to load power actor: %w", err)
	}

	ract, err := m.StateGetActor(ctx, reward.Address, tsk)
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("failed to load reward actor: %w", err)
	}

	pst, err := power.Load(m.StateManager.ChainStore().ActorStore(ctx), pact)
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("failed to load power actor state: %w", err)
	}

	rst, err := reward.Load(m.StateManager.ChainStore().ActorStore(ctx), ract)
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("failed to load reward actor state: %w", err)
	}

	circ, err := stateVMCirculatingSupplyInternal(ctx, ts.Key(), m.Chain, m.StateManager)
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("getting total circulating supply: %w", err)
	}

	powClaim, err := pst.TotalPower()
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("getting total power: %w", err)
	}

	rewPow, err := rst.ThisEpochBaselinePower()
	if err != nil {
		return api.DealCollateralBounds{}, xerrors.Errorf("getting reward baseline power: %w", err)
	}

	min, max := policy.DealProviderCollateralBounds(size,
		verified,
		powClaim.RawBytePower,
		powClaim.QualityAdjPower,
		rewPow,
		circ.EpkCirculating,
		m.StateManager.GetNtwkVersion(ctx, ts.Height()))
	return api.DealCollateralBounds{
		Min: types.BigDiv(types.BigMul(min, dealProviderCollateralNum), dealProviderCollateralDen),
		Max: max,
	}, nil
} */

func (a *StateAPI) StateTotalMinedDetail(ctx context.Context, tsk types.TipSetKey) (*reward.TotalMinedDetail, error) {
	act, err := a.StateGetActor(ctx, reward.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load reward actor state: %w", err)
	}

	rst, err := reward.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, err
	}
	return rst.TotalMinedDetail()
}

func (a *StateAPI) StateCirculatingSupply(ctx context.Context, tsk types.TipSetKey) (abi.TokenAmount, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return types.EmptyInt, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	sTree, err := a.StateManager.ParentState(ts)
	if err != nil {
		return types.EmptyInt, err
	}
	return a.StateManager.GetCirculatingSupply(ctx, ts.Height()-1, sTree)
}

func (a *StateAPI) StateVMCirculatingSupplyInternal(ctx context.Context, tsk types.TipSetKey) (api.CirculatingSupply, error) {
	return stateVMCirculatingSupplyInternal(ctx, tsk, a.Chain, a.StateManager)
}

func stateVMCirculatingSupplyInternal(
	ctx context.Context,
	tsk types.TipSetKey,
	cstore *store.ChainStore,
	smgr *stmgr.StateManager,
) (api.CirculatingSupply, error) {
	ts, err := cstore.GetTipSetFromKey(tsk)
	if err != nil {
		return api.CirculatingSupply{}, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	sTree, err := smgr.ParentState(ts)
	if err != nil {
		return api.CirculatingSupply{}, err
	}

	return smgr.GetVMCirculatingSupplyDetailed(ctx, ts.Height(), sTree)
}

func (m *StateModule) StateNetworkVersion(ctx context.Context, tsk types.TipSetKey) (network.Version, error) {
	ts, err := m.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return network.VersionMax, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}

	// TODO: Height-1 to be consistent with the rest of the APIs?
	// But that's likely going to break a bunch of stuff.
	return m.StateManager.GetNtwkVersion(ctx, ts.Height()), nil
}

func (a *StateAPI) StateListExperts(ctx context.Context, tsk types.TipSetKey) ([]address.Address, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	return stmgr.ListExpertActors(ctx, a.StateManager, ts)
}

func (a *StateAPI) StateExpertInfo(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.ExpertInfo, error) {
	if addr.Protocol() != address.ID {
		return nil, xerrors.Errorf("not a ID address: %s", addr)
	}
	act, err := a.StateGetActor(ctx, addr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expert actor: %w", err)
	}

	// expertfund
	efAct, err := a.StateGetActor(ctx, builtin.ExpertFundActorAddr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expertfund actor: %w", err)
	}

	efs, err := expertfund.Load(a.Chain.ActorStore(ctx), efAct)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expertfund actor state: %w", err)
	}

	efInfo, err := efs.ExpertInfo(addr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get expertfund info: %w", err)
	}
	defInfo, err := efs.DisqualifiedExpertInfo(addr)
	if err != nil {
		return nil, xerrors.Errorf("failed to get disqualification info: %w", err)
	}

	// expert
	eas, err := expert.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expert actor state: %w", err)
	}

	info, err := eas.Info()
	if err != nil {
		return nil, err
	}

	switch info.Status {
	case expert2.ExpertStateRegistered:
		info.StatusDesc = "registered"
	case expert2.ExpertStateUnqualified:
		if defInfo != nil {
			info.StatusDesc = fmt.Sprintf("unqualified (no enough votes, disqualified at %d)", defInfo.DisqualifiedAt)
			info.LostEpoch = defInfo.DisqualifiedAt
		} else {
			info.StatusDesc = "no enough votes"
			info.LostEpoch = -1
		}
	case expert2.ExpertStateQualified:
		info.StatusDesc = "qualified"
	case expert2.ExpertStateBlocked:
		info.StatusDesc = "blocked"
	default:
		return nil, xerrors.Errorf("unknow expert status: %d", info.Status)
	}

	return &api.ExpertInfo{
		ExpertInfo:  *info,
		TotalReward: efInfo.RewardDebt,
	}, nil
}

func (a *StateAPI) StateExpertDatas(ctx context.Context, addr address.Address, filter *bitfield.BitField, filterOut bool, tsk types.TipSetKey) ([]*expert.DataOnChainInfo, error) {
	act, err := a.StateGetActor(ctx, addr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expert actor: %w", err)
	}

	eas, err := expert.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expert actor state: %w", err)
	}

	return eas.Datas()
}

func (a *StateAPI) StateExpertFileInfo(ctx context.Context, pieceCid cid.Cid, tsk types.TipSetKey) (*api.ExpertFileInfo, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, expertfund.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expertfund actor: %w", err)
	}

	st, err := expertfund.Load(a.StateManager.ChainStore().ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expertfund actor state: %w", err)
	}

	expertAddr, err := st.DataExpert(pieceCid)
	if err != nil {
		return nil, err
	}

	expertAct, err := a.StateManager.LoadActorTsk(ctx, expertAddr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expert actor: %w", err)
	}

	expertSt, err := expert.Load(a.StateManager.ChainStore().ActorStore(ctx), expertAct)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expert actor state: %w", err)
	}

	data, err := expertSt.Data(pieceCid)
	if err != nil {
		return nil, err
	}
	return &api.ExpertFileInfo{
		Expert:     expertAddr,
		PieceID:    pieceCid,
		PieceSize:  data.PieceSize,
		Redundancy: data.Redundancy,
	}, nil
}

func (a *StateAPI) StateVoteTally(ctx context.Context, tsk types.TipSetKey) (*vote.Tally, error) {
	act, err := a.StateGetActor(ctx, vote.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to get vote actor: %w", err)
	}

	vst, err := vote.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load vote actor state: %w", err)
	}
	return vst.Tally()
}

func (a *StateAPI) StateVoterInfo(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*vote.VoterInfo, error) {
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to get tipset: %w", err)
	}

	sTree, err := a.StateManager.ParentState(ts)
	if err != nil {
		return nil, xerrors.Errorf("failed to get tipset state: %w", err)
	}

	act, err := sTree.GetActor(vote.Address)
	if err != nil {
		return nil, xerrors.Errorf("failed to load vote actor: %w", err)
	}

	vst, err := vote.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load vote actor state: %w", err)
	}

	ida, err := a.StateManager.LookupID(ctx, addr, ts)
	if err != nil {
		return nil, xerrors.Errorf("failed to look up id for %s: %w", addr, err)
	}

	vAct, err := sTree.GetActor(builtin.VoteFundActorAddr)
	if err != nil {
		return nil, xerrors.Errorf("failed to load votefund actor: %w", err)
	}

	return vst.VoterInfo(ida, ts.Height(), vAct.Balance)
}

func (a *StateAPI) StateKnowledgeInfo(ctx context.Context, tsk types.TipSetKey) (*knowledge.Info, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, knowledge.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load power actor: %w", err)
	}

	knoState, err := knowledge.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load knowledge actor state: %w", err)
	}
	return knoState.Info()
}

func (a *StateAPI) StateGovernSupervisor(ctx context.Context, tsk types.TipSetKey) (address.Address, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, govern.Address, tsk)
	if err != nil {
		return address.Undef, xerrors.Errorf("failed to load govern actor: %w", err)
	}

	govState, err := govern.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return address.Undef, xerrors.Errorf("failed to load govern actor state: %w", err)
	}
	return govState.Supervior(), nil
}

func (a *StateAPI) StateGovernorList(ctx context.Context, tsk types.TipSetKey) ([]*govern.GovernorInfo, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, govern.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load govern actor: %w", err)
	}

	govState, err := govern.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load govern actor state: %w", err)
	}
	return govState.ListGovrnors()
}

func (a *StateAPI) StateGovernParams(ctx context.Context, tsk types.TipSetKey) (*govern.GovParams, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, govern.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load govern actor: %w", err)
	}
	store := a.Chain.ActorStore(ctx)

	var out govern.GovParams

	// ratio
	powerState, err := power.Load(store, act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load power actor state: %w", err)
	}
	out.MinersPoStRatio, err = powerState.PoStRatio()
	if err != nil {
		return nil, xerrors.Errorf("failed to get ratio")
	}

	// quota
	marketState, err := market.Load(store, act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load market actor state: %w", err)
	}
	quotas, err := marketState.Quotas()
	if err != nil {
		return nil, xerrors.Errorf("failed to get quotas: %w", err)
	}
	out.MarketInitialQuota = quotas.InitialQuota()

	// knowledge
	knowState, err := knowledge.Load(store, act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load knowledge actor state: %w", err)
	}
	info, err := knowState.Info()
	if err != nil {
		return nil, xerrors.Errorf("failed to get knowledge fund info: %w", err)
	}
	out.KnowledgePayee = info.Payee

	// threshold
	efState, err := expertfund.Load(store, act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load expertfund actor state: %w", err)
	}
	out.ExpertfundThreshold = efState.Threshold()

	return &out, nil
}

func (a *StateAPI) StateRetrievalInfo(ctx context.Context, tsk types.TipSetKey) (*api.RetrievalInfo, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, retrieval.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval actor: %w", err)
	}

	state, err := retrieval.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval actor state: %w", err)
	}
	pledge, err := state.TotalCollateral()
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval TotalCollateral: %w", err)
	}

	reward, err := state.TotalRetrievalReward()
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval total reward: %w", err)
	}
	pending, err := state.PendingReward()
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval pending reward: %w", err)
	}
	return &api.RetrievalInfo{
		TotalPledge:   pledge,
		TotalReward:   reward,
		PendingReward: pending,
	}, nil
}

func (a *StateAPI) StateRetrievalPledge(ctx context.Context, addr address.Address, tsk types.TipSetKey) (*api.RetrievalState, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, retrieval.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval actor: %w", err)
	}

	ida, err := a.StateLookupID(ctx, addr, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to lookup id: %w", err)
	}

	state, err := retrieval.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval actor state: %w", err)
	}
	info, err := state.StateInfo(ida)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval state info: %w", err)
	}
	ts, err := a.Chain.GetTipSetFromKey(tsk)
	if err != nil {
		return nil, xerrors.Errorf("loading tipset %s: %w", tsk, err)
	}
	expend, err := state.DayExpend(ts.Height(), ida)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval expend: %w", err)
	}
	var locked retrieval.LockedState
	found, err := state.LockedState(ida, &locked)
	if err != nil {
		return nil, xerrors.Errorf("failed to load retrieval locked: %w", err)
	}
	ret := &api.RetrievalState{
		BindMiners:  info.BindMiners,
		Balance:     info.Amount,
		DayExpend:   expend,
		Locked:      abi.NewTokenAmount(0),
		LockedEpoch: abi.ChainEpoch(0),
	}
	if found {
		ret.Locked = locked.Amount
		ret.LockedEpoch = locked.ApplyEpoch
	}
	return ret, nil
}

func (a *StateAPI) StateDataIndex(ctx context.Context, epoch abi.ChainEpoch, tsk types.TipSetKey) ([]*api.DataIndex, error) {
	act, err := a.StateManager.LoadActorTsk(ctx, market.Address, tsk)
	if err != nil {
		return nil, xerrors.Errorf("failed to load market actor: %w", err)
	}

	state, err := market.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return nil, xerrors.Errorf("failed to load market actor state: %w", err)
	}

	dataIndex, err := state.DataIndexes()
	if err != nil {
		return nil, xerrors.Errorf("failed to load state index: %w", err)
	}

	var ret []*api.DataIndex
	err = dataIndex.ForEach(epoch, func(provider address.Address, data market.DataIndex) error {
		root, err := cid.Parse(data.RootCID)
		if err != nil {
			return err
		}
		ret = append(ret, &api.DataIndex{
			Miner:    provider,
			RootCID:  root,
			PieceCID: data.PieceCID,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (a *StateAPI) StateMinerStoredAnyPiece(ctx context.Context, maddr address.Address, pieceIDs []cid.Cid, tsk types.TipSetKey) (bool, error) {

	// check pending
	mact, err := a.StateManager.LoadActorTsk(ctx, builtin.StorageMarketActorAddr, tsk)
	if err != nil {
		return false, xerrors.Errorf("failed to load market actor: %w", err)
	}

	mstate, err := market.Load(a.Chain.ActorStore(ctx), mact)
	if err != nil {
		return false, xerrors.Errorf("failed to load market actor state: %w", err)
	}

	has, err := mstate.HasPendingPiece(maddr, pieceIDs)
	if err != nil {
		return false, xerrors.Errorf("failed to check pending deals: %w", err)
	}
	if has {
		return true, nil
	}

	// check miner
	act, err := a.StateManager.LoadActorTsk(ctx, maddr, tsk)
	if err != nil {
		return false, xerrors.Errorf("failed to load miner actor: %w", err)
	}

	state, err := miner.Load(a.Chain.ActorStore(ctx), act)
	if err != nil {
		return false, xerrors.Errorf("failed to load miner actor state: %w", err)
	}

	return state.ContainsAnyPiece(pieceIDs)
}
