package miner

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/chain/types"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/retrievalmarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-state-types/abi"
	lru "github.com/hashicorp/golang-lru"
	cid "github.com/ipfs/go-cid"
	"go.opencensus.io/trace"
)

var (
	//LoopWaitingSeconds data check loop waiting seconds
	LoopWaitingSeconds = time.Second * 30
	// RetrieveParallelNum num
	RetrieveParallelNum = 32
	// DealParallelNum deal thread parallel num
	DealParallelNum = 32
	// RetrieveTryCountMax retrieve try count max
	RetrieveTryCountMax = 50
)

type DataRef struct {
	pieceID     cid.Cid
	rootCID     cid.Cid
	miners      []address.Address
	tryCount    int
	isRetrieved bool
	isDealed    bool
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

	totalDataCount     uint64
	totalRetrieveCount uint64
	totalDealCount     uint64
}

func newMinerData(api api.FullNode, addr address.Address) *MinerData {
	data, err := lru.NewARC(1000000)
	if err != nil {
		panic(err)
	}
	retrievals, err := lru.NewARC(1000)
	if err != nil {
		panic(err)
	}
	deals, err := lru.NewARC(1000)
	if err != nil {
		panic(err)
	}
	return &MinerData{
		api:                api,
		address:            addr,
		dataRefs:           data,
		retrievals:         retrievals,
		deals:              deals,
		checkHeight:        10,
		totalDataCount:     0,
		totalRetrieveCount: 0,
		totalDealCount:     0,
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

		if err := m.checkChainData(ctx); err != nil {
			log.Errorf("failed to check chain data: %s", err)
		}

		if err := m.retrieveChainData(ctx); err != nil {
			log.Warnf("failed to retrieve data: %s", err)
		}

		if err := m.dealChainData(ctx); err != nil {
			log.Errorf("failed to deal chain data: %s", err)
		}
		log.Infof("sync data height:%d, data:%d, retrieve:%d, deal:%d, wait deal:%d", m.checkHeight, m.totalDataCount, m.totalRetrieveCount, m.totalDealCount, m.dataRefs.Len())
		m.niceSleep(LoopWaitingSeconds)
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

func (m *MinerData) checkChainData(ctx context.Context) error {
	head, err := m.api.ChainHead(ctx)
	if err != nil {
		return err
	}

	for m.checkHeight < head.Height() {
		if m.stopping != nil {
			break
		}

		datas, err := m.api.StateDataIndex(ctx, m.checkHeight, types.EmptyTSK)
		if err != nil {
			return err
		}
		for _, data := range datas {
			var dataRef *DataRef
			ref, ok := m.dataRefs.Get(data.PieceCID.String())
			if !ok {
				dataRef = &DataRef{
					pieceID:     data.PieceCID,
					rootCID:     data.RootCID,
					miners:      []address.Address{},
					isRetrieved: false,
					isDealed:    false,
				}
				m.totalDataCount++
			} else {
				dataRef = ref.(*DataRef)
			}
			dataRef.miners = append(dataRef.miners, data.Miner)
			m.dataRefs.Add(data.PieceCID.String(), dataRef)
		}

		m.checkHeight++
	}
	return nil
}

func (m *MinerData) retrieveChainData(ctx context.Context) error {
	// check retrieve deals state
	retrieveKeys := m.retrievals.Keys()
	for _, rk := range retrieveKeys {
		dealObj, _ := m.retrievals.Get(rk)
		deal := dealObj.(*api.RetrievalDeal)

		nDeal, err := m.api.ClientRetrieveGetDeal(ctx, deal.DealID)
		if err != nil {
			return err
		}

		if retrievalmarket.IsTerminalSuccess(nDeal.Status) {
			dataObj, _ := m.dataRefs.Get(rk)
			data := dataObj.(*DataRef)
			data.isRetrieved = true
			m.totalRetrieveCount++
		}
		if retrievalmarket.IsTerminalStatus(nDeal.Status) {
			m.retrievals.Remove(rk)
		}
	}
	if m.retrievals.Len() >= RetrieveParallelNum {
		log.Infof("wait for retrieval:%d", m.retrievals.Len())
		return nil
	}

	deals, err := m.api.ClientRetrieveListDeals(ctx)
	if err != nil {
		return err
	}

	keys := m.dataRefs.Keys()
	for _, rk := range keys {
		dataObj, _ := m.dataRefs.Get(rk)
		data := dataObj.(*DataRef)

		for _, d := range deals {
			if d.PieceCID.Equals(data.pieceID) {
				if retrievalmarket.IsTerminalSuccess(d.Status) {
					data.isRetrieved = true
					m.totalRetrieveCount++
				}
				if !retrievalmarket.IsTerminalStatus(d.Status) {
					m.retrievals.Add(rk, d)
				}
				break
			}
		}

		if data.isRetrieved {
			continue
		}
		if m.retrievals.Contains(data.pieceID.String()) {
			continue
		}

		if m.retrievals.Len() >= RetrieveParallelNum {
			log.Infof("wait for retrieval:%d", m.retrievals.Len())
			break
		}

		miner := data.miners[rand.Intn(len(data.miners))]
		deal, err := m.api.ClientRetrieveQuery(ctx, data.rootCID, &data.pieceID, miner)
		if err != nil {
			data.tryCount++
			log.Warnf("failed to retrieve miner:%s, data:%s, try:%d, err:%s", miner, data.rootCID, data.tryCount, err)
			// if data.tryCount > RetrieveTryCountMax {
			// 	for index, m := range data.miners {
			// 		if m == miner {
			// 			data.miners = append(data.miners[:index], data.miners[index+1:]...)
			// 			break
			// 		}
			// 	}
			// 	m.dataRefs.Add(rk, data)
			// }
			continue
		}
		log.Warnf("client retrieve miner:%s, data:%s", miner, data.rootCID)

		m.retrievals.Add(rk, deal)
	}
	return nil
}

func checkDealStatus(deal *api.DealInfo) (bool, bool) {
	isDealed := (deal.State == storagemarket.StorageDealAwaitingPreCommit ||
		deal.State == storagemarket.StorageDealActive)
	isError := (deal.State == storagemarket.StorageDealProposalNotFound ||
		deal.State == storagemarket.StorageDealProposalRejected ||
		deal.State == storagemarket.StorageDealFailing ||
		deal.State == storagemarket.StorageDealError)
	return isDealed || isError, isDealed
}

func (m *MinerData) dealChainData(ctx context.Context) error {
	dealKeys := m.deals.Keys()
	for _, rk := range dealKeys {
		id, _ := m.deals.Get(rk)
		dealID := id.(cid.Cid)
		deal, err := m.api.ClientGetDealInfo(ctx, dealID)
		if err != nil {
			return err
		}
		isFinish, isDealed := checkDealStatus(deal)
		if isDealed {
			dataObj, _ := m.dataRefs.Get(rk)
			data := dataObj.(*DataRef)
			data.isDealed = true
			m.totalDealCount++
		}
		if isFinish {
			m.deals.Remove(rk)
		}
	}
	if m.deals.Len() >= DealParallelNum {
		log.Infof("wait for deal:%d", m.deals.Len())
		return nil
	}

	deals, err := m.api.ClientListDeals(ctx)
	if err != nil {
		return err
	}
	keys := m.dataRefs.Keys()
	for _, rk := range keys {
		dataObj, _ := m.dataRefs.Get(rk)
		data := dataObj.(*DataRef)

		// if data not found local, go to next one
		if !data.isRetrieved {
			continue
		}

		for _, d := range deals {
			if d.PieceCID.Equals(data.pieceID) {
				isFinish, isDealed := checkDealStatus(&d)
				if isDealed {
					data.isDealed = true
					m.totalDealCount++
				}
				if !isFinish {
					m.deals.Add(rk, d.ProposalCid)
				}
				break
			}
		}

		if data.isDealed {
			continue
		}
		// if miner is dealing, go to next one
		if m.deals.Contains(rk) {
			continue
		}

		if m.deals.Len() >= DealParallelNum {
			log.Infof("wait for deal:%d", m.deals.Len())
			break
		}

		// mi, err := m.api.StateMinerInfo(ctx, m.address, types.EmptyTSK)
		// if err != nil {
		// 	return err
		// }

		// if *mi.PeerId == peer.ID("SETME") {
		// 	return fmt.Errorf("the miner hasn't initialized yet")
		// }

		/* ask, err := m.api.ClientQueryAsk(ctx, *mi.PeerId, m.address)
		if err != nil {
			return err
		} */

		stData := &storagemarket.DataRef{
			TransferType: storagemarket.TTGraphsync,
			Root:         data.rootCID,
		}
		params := &api.StartDealParams{
			Data:   stData,
			Wallet: address.Undef,
			Miner:  m.address,
			/* EpochPrice:        ask.Price,
			MinBlocksDuration: uint64(ask.Expiry - ts.Height()), */
			FastRetrieval: true,
		}
		dealID, err := m.api.ClientStartDeal(ctx, params)
		if err != nil {
			log.Errorf("failed to start deal: %s", err)
			continue
		}
		log.Warnf("start deal with miner:%s deal: %s", m.address, dealID.String())
		m.deals.Add(rk, *dealID)
	}
	return nil
}
