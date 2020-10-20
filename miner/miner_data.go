package miner

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	minerActor "github.com/filecoin-project/specs-actors/actors/builtin/miner"
	lru "github.com/hashicorp/golang-lru"
	cid "github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"go.opencensus.io/trace"
	"golang.org/x/xerrors"
)

type dealData struct {
	dealID  abi.DealID
	deal    market.DealProposal
	dataRef market.PublishStorageDataRef
	state   market.DealState
}

type MinerData struct {
	api api.FullNode

	lk      sync.Mutex
	address address.Address

	stop     chan struct{}
	stopping chan struct{}

	checkHeight abi.ChainEpoch

	dataRefs   *lru.ARCCache
	retrievals *lru.ARCCache
	deals      *lru.ARCCache
}

func newMinerData(api api.FullNode, addr address.Address) *MinerData {
	data, err := lru.NewARC(10000)
	if err != nil {
		panic(err)
	}
	retrievals, err := lru.NewARC(10000)
	if err != nil {
		panic(err)
	}
	deals, err := lru.NewARC(10000)
	if err != nil {
		panic(err)
	}
	return &MinerData{
		api:        api,
		address:    addr,
		dataRefs:   data,
		retrievals: retrievals,
		deals:      deals,
	}
}

func (m *MinerData) Start(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()
	if m.stop != nil {
		return fmt.Errorf("miner data already started")
	}
	m.stop = make(chan struct{})
	go m.syncData(context.TODO())
	return nil
}

func (m *MinerData) Stop(ctx context.Context) error {
	m.lk.Lock()
	defer m.lk.Unlock()

	m.stopping = make(chan struct{})
	stopping := m.stopping
	close(m.stop)

	select {
	case <-stopping:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *MinerData) syncData(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "/mine/sync")
	defer span.End()

	for {
		select {
		case <-m.stop:
			stopping := m.stopping
			m.stop = nil
			m.stopping = nil
			close(stopping)
			return

		default:
		}

		needCheck, err := m.needCheckData(ctx)
		if err != nil {
			log.Errorf("failed to call need data: %s", err)
			continue
		}

		if needCheck {
			if err := m.checkChainData(ctx); err != nil {
				log.Errorf("failed to check chain data: %s", err)
			}
		}

		if err := m.retrieveChainData(ctx); err != nil {
			log.Warnf("failed to retrieve data: %s", err)
		}

		if err := m.dealChainData(ctx); err != nil {
			log.Errorf("failed to deal chain data: %s", err)
		}
		m.niceSleep(time.Second * 5)
	}
}

func (m *MinerData) niceSleep(d time.Duration) bool {
	select {
	case <-time.After(d):
		return true
	case <-m.stop:
		return false
	}
}

func (m *MinerData) needCheckData(ctx context.Context) (bool, error) {
	sync, err := m.api.SyncState(ctx)
	if err != nil {
		return false, err
	}
	for _, ss := range sync.ActiveSyncs {
		var heightDiff int64
		if ss.Base != nil {
			heightDiff = int64(ss.Base.Height())
		}
		if ss.Target != nil {
			heightDiff = int64(ss.Target.Height()) - heightDiff
		} else {
			heightDiff = 0
		}
		if heightDiff > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (m *MinerData) checkChainData(ctx context.Context) error {
	head, err := m.api.ChainHead(ctx)
	if err != nil {
		return err
	}
	localDeals, err := m.api.ClientListDeals(ctx)
	if err != nil {
		return err
	}

	for m.checkHeight < head.Height() {
		tipset, err := m.api.ChainGetTipSetByHeight(ctx, m.checkHeight, types.EmptyTSK)
		if err != nil {
			return err
		}
		messages, err := m.api.ChainGetParentMessages(ctx, tipset.Cids()[0])
		for _, msg := range messages {
			if msg.Message.To == builtin.StorageMarketActorAddr &&
				msg.Message.Method == builtin.MethodsMarket.PublishStorageDeals {
				var params market.PublishStorageDealsParams
				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Message.Params)); err != nil {
					return err
				}

				for _, deal := range params.Deals {
					if ok, _ := m.isMinerDealed(ctx, params.DataRef.RootCID, &deal.Proposal, localDeals); ok {
						continue
					}
					dealData := &dealData{
						deal:    deal.Proposal,
						dataRef: params.DataRef,
					}
					m.dataRefs.Add(dealData.deal.PieceCID.String(), dealData)
				}
			} else if msg.Message.Method == builtin.MethodsMiner.PreCommitSector {
				var params minerActor.SectorPreCommitInfo
				if err := params.UnmarshalCBOR(bytes.NewReader(msg.Message.Params)); err != nil {
					return xerrors.Errorf("unmarshal pre commit: %w", err)
				}

				for _, did := range params.DealIDs {
					deal, err := m.api.StateMarketStorageDeal(ctx, did, types.EmptyTSK)
					if err != nil {
						return err
					}
					if ok, _ := m.isMinerDealed(ctx, cid.Undef, &deal.Proposal, localDeals); ok {
						continue
					}
					data, ok := m.dataRefs.Get(deal.Proposal.PieceCID.String())
					if !ok {
						return fmt.Errorf("the deal data not found:%s", deal.Proposal.PieceCID.String())
					}
					dealData := data.(dealData)
					dealData.dealID = did
					dealData.state = deal.State
				}
			}
		}
		m.checkHeight++
	}
	return nil
}

func (m *MinerData) isMinerDealed(ctx context.Context, root cid.Cid, deal *market.DealProposal, localDeals []api.DealInfo) (bool, error) {
	if deal.Provider == m.address {
		return true, nil
	}
	if root != cid.Undef {
		offer, err := m.api.ClientMinerQueryOffer(ctx, root, m.address)
		if err != nil {
			return false, err
		}
		// if miner has sealed the data, return true
		if offer.Err == "" {
			return true, nil
		}
	}

	for _, lDeal := range localDeals {
		if lDeal.PieceCID == deal.PieceCID &&
			lDeal.State != storagemarket.StorageDealProposalNotFound &&
			lDeal.State != storagemarket.StorageDealNotFound &&
			lDeal.State != storagemarket.StorageDealProposalRejected &&
			lDeal.State != storagemarket.StorageDealFailing &&
			lDeal.State != storagemarket.StorageDealError {
			return true, nil
		}
	}

	return false, nil
}

func (m *MinerData) retrieveChainData(ctx context.Context) error {
	keys := m.dataRefs.Keys()
	for _, rk := range keys {
		data, _ := m.dataRefs.Get(rk)
		dealData := data.(dealData)

		if dealData.dealID > 0 && dealData.state.SectorStartEpoch > 0 {
			has, err := m.api.ClientHasLocal(ctx, dealData.dataRef.RootCID)
			if err != nil {
				return err
			}

			if has {
				m.retrievals.Remove(rk)
				continue
			}

			if m.retrievals.Contains(dealData.deal.PieceCID.String()) {
				continue
			}

			resp, err := m.api.ClientQuery(ctx, dealData.dataRef.RootCID)
			if err != nil {
				log.Warnf("failed to query data:%s,err:%s", dealData.dataRef.RootCID, err)
				continue
			}
			if resp.Status == api.QuerySuccess {
				m.retrievals.Remove(rk)
			} else {
				m.retrievals.Add(rk, resp)
			}
		}

		if m.retrievals.Len() > 3 {
			break
		}
	}
	return nil
}

func (m *MinerData) dealChainData(ctx context.Context) error {
	keys := m.dataRefs.Keys()
	for _, rk := range keys {
		data, _ := m.dataRefs.Get(rk)
		dealData := data.(dealData)

		has, err := m.api.ClientHasLocal(ctx, dealData.dataRef.RootCID)
		if err != nil {
			return err
		}

		// if data not found local, go to next one
		if !has {
			continue
		}

		if m.deals.Contains(rk) {
			continue
		}

		offer, err := m.api.ClientMinerQueryOffer(ctx, dealData.dataRef.RootCID, m.address)
		if err != nil {
			return err
		}
		// if miner has sealed the data, go to next one
		if offer.Err == "" {
			m.dataRefs.Remove(rk)
			m.deals.Remove(rk)
			continue
		}

		// if miner is dealing, go to next one
		if m.deals.Contains(rk) {
			continue
		}

		ts, err := m.api.ChainHead(ctx)
		if err != nil {
			return err
		}

		mi, err := m.api.StateMinerInfo(ctx, m.address, types.EmptyTSK)
		if err != nil {
			return err
		}

		if peer.ID(mi.PeerId) == peer.ID("SETME") {
			return fmt.Errorf("the miner hasn't initialized yet")
		}

		pid := peer.ID(mi.PeerId)
		ask, err := m.api.ClientQueryAsk(ctx, pid, m.address)
		if err != nil {
			return err
		}

		offers, err := m.api.ClientFindData(ctx, dealData.dataRef.RootCID)
		if err != nil {
			return err
		}

		stData := &storagemarket.DataRef{
			TransferType: storagemarket.TTGraphsync,
			Root:         dealData.dataRef.RootCID,
			Expert:       dealData.dataRef.Expert,
			Bounty:       dealData.dataRef.Bounty,
		}
		params := &api.StartDealParams{
			Data:              stData,
			Wallet:            address.Undef,
			Miner:             m.address,
			EpochPrice:        ask.Ask.Price,
			MinBlocksDuration: uint64(ask.Ask.Expiry - ts.Height()),
			Redundancy:        int64(len(offers)),
		}
		dealID, err := m.api.ClientStartDeal(ctx, params)
		if err != nil {
			log.Errorf("failed to start deal: %s", err)
			continue
		}
		log.Warnf("start miner:%s deal: %s", m.address, dealID.String())

		m.deals.Add(rk, dealID.String())

		if m.deals.Len() > 3 {
			break
		}
	}
	return nil
}
