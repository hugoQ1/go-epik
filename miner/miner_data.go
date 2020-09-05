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
	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"go.opencensus.io/trace"
)

type MinerData struct {
	api api.FullNode

	lk      sync.Mutex
	address address.Address

	stop     chan struct{}
	stopping chan struct{}

	checkHeight abi.ChainEpoch

	dataRefs *lru.ARCCache
}

func newMinerData(api api.FullNode, addr address.Address) *MinerData {
	arc, err := lru.NewARC(10000)
	if err != nil {
		panic(err)
	}
	return &MinerData{
		api:      api,
		address:  addr,
		dataRefs: arc,
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
			m.niceSleep(time.Second * 5)
			continue
		}

		if needCheck {
			if err := m.checkChainData(ctx); err != nil {
				log.Errorf("failed to check chain data: %s", err)
				m.niceSleep(time.Second * 5)
			}
		} else {
			m.niceSleep(time.Second * 5)
		}

		if err := m.retrieveChainData(ctx); err != nil {
			log.Errorf("failed to retrieve data: %s", err)
			m.niceSleep(time.Second * 5)
		}

		if err := m.dealChainData(ctx); err != nil {
			log.Errorf("failed to deal chain data: %s", err)
			m.niceSleep(time.Second * 5)
		}
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
				offer, err := m.api.ClientMinerQueryOffer(ctx, params.DataRef.RootCID, m.address)
				if err != nil {
					return err
				}
				if offer.Err != "" {
					m.dataRefs.Add(params.DataRef.RootCID.String(), params.DataRef)
				}
			}
		}
		m.checkHeight++
	}
	return nil
}

func (m *MinerData) retrieveChainData(ctx context.Context) error {
	for m.dataRefs.Len() > 0 {
		keys := m.dataRefs.Keys()
		for _, rk := range keys {
			data, _ := m.dataRefs.Get(rk)
			dataRef := data.(market.PublishStorageDataRef)

			has, err := m.api.ClientHasLocal(ctx, dataRef.RootCID)
			if err != nil {
				return err
			}
			if has {
				continue
			}

			if _, err := m.api.ClientQuery(ctx, []cid.Cid{dataRef.RootCID}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MinerData) dealChainData(ctx context.Context) error {
	for m.dataRefs.Len() > 0 {
		keys := m.dataRefs.Keys()
		for _, rk := range keys {
			data, _ := m.dataRefs.Get(rk)
			dataRef := data.(market.PublishStorageDataRef)

			has, err := m.api.ClientHasLocal(ctx, dataRef.RootCID)
			if err != nil {
				return err
			}

			// if data not found local, go to next one
			if !has {
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

			offers, err := m.api.ClientFindData(ctx, dataRef.RootCID)
			if err != nil {
				return err
			}

			stData := &storagemarket.DataRef{
				Root:   dataRef.RootCID,
				Expert: dataRef.Expert,
				Bounty: dataRef.Bounty,
			}
			params := &api.StartDealParams{
				Data:              stData,
				Wallet:            address.Undef,
				Miner:             m.address,
				EpochPrice:        ask.Ask.Price,
				MinBlocksDuration: uint64(ask.Ask.Expiry - ts.Height()),
				Redundancy:        int64(len(offers)),
			}
			_, err = m.api.ClientStartDeal(ctx, params)
			if err != nil {
				return err
			}
			m.dataRefs.Remove(dataRef.RootCID.String())
		}
	}
	return nil
}
