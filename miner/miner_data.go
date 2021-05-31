package miner

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/EpiK-Protocol/go-epik/api"
	mineractor "github.com/EpiK-Protocol/go-epik/chain/actors/builtin/miner"
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

	// MinerDefaultScore default score
	MinerDefaultScore = 8

	// MinerMaxScore max score
	MinerMaxScore = 10

	// MinerPunishmentScore
	MinerPunishmentScore = 2
)

type DataRef struct {
	pieceID      cid.Cid
	rootCID      cid.Cid
	miners       map[address.Address]int
	tryCount     int
	retrieveTime time.Time
	isRetrieved  bool
	isDealed     bool
}

type MinerData struct {
	api api.FullNode

	lk        sync.Mutex
	miner     address.Address
	minerInfo mineractor.MinerInfo

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
	return &MinerData{
		api:                api,
		miner:              addr,
		dataRefs:           data,
		retrievals:         nil,
		deals:              nil,
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

		if err := m.storageChainData(ctx); err != nil {
			log.Errorf("failed to deal chain data: %s", err)
		}
		log.Infof("sync data height:%d, data:%d, retrieved:%d, storaged:%d, retrievaling:%d, dealing:%d", m.checkHeight, m.totalDataCount, m.totalRetrieveCount, m.totalDealCount, m.retrievals.Len(), m.deals.Len())
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
					miners:      make(map[address.Address]int),
					isRetrieved: false,
					isDealed:    false,
				}
				m.totalDataCount++
			} else {
				dataRef = ref.(*DataRef)
			}
			dataRef.miners[data.Miner] = MinerDefaultScore
			m.dataRefs.Add(data.PieceCID.String(), dataRef)
		}

		m.checkHeight++
	}
	return nil
}

func (m *MinerData) loadRetrievalData(ctx context.Context) (*lru.ARCCache, error) {
	info, err := m.api.StateMinerInfo(ctx, m.miner, types.EmptyTSK)
	if err != nil {
		return nil, err
	}
	m.minerInfo = info
	retrievals, err := lru.NewARC(RetrieveParallelNum * 2)
	if err != nil {
		return nil, err
	}

	deals, err := m.api.ClientRetrieveListDeals(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range deals {
		if retrievalmarket.IsTerminalSuccess(d.Status) {
			dataObj, ok := m.dataRefs.Get(d.PieceCID.String())
			if ok {
				data := dataObj.(*DataRef)
				data.isRetrieved = true
				m.totalRetrieveCount++
			}
		}
		if !(d.Status == retrievalmarket.DealStatusErrored ||
			d.Status == retrievalmarket.DealStatusCancelled ||
			retrievalmarket.IsTerminalStatus(d.Status)) {
			retrievals.Add(d.PieceCID.String(), d)
		}
	}
	return retrievals, nil
}

func (m *MinerData) retrieveChainData(ctx context.Context) error {
	if m.retrievals == nil {
		data, err := m.loadRetrievalData(ctx)
		if err != nil {
			return err
		}
		m.retrievals = data
	}
	// check retrieve deals state
	retrieveKeys := m.retrievals.Keys()
	for _, rk := range retrieveKeys {
		dealObj, _ := m.retrievals.Get(rk)
		deal := dealObj.(*api.RetrievalDeal)

		nDeal, err := m.api.ClientRetrieveGetDeal(ctx, deal.DealID)
		if err != nil {
			return err
		}

		dataObj, _ := m.dataRefs.Get(rk)
		data := dataObj.(*DataRef)
		if !data.isRetrieved && retrievalmarket.IsTerminalSuccess(nDeal.Status) {
			data.isRetrieved = true
			m.totalRetrieveCount++
		}
		if nDeal.Status == retrievalmarket.DealStatusErrored ||
			nDeal.Status == retrievalmarket.DealStatusCancelled ||
			retrievalmarket.IsTerminalStatus(nDeal.Status) {
			m.retrievals.Remove(rk)
			if !retrievalmarket.IsTerminalSuccess(nDeal.Status) {
				data.tryCount++
				if _, ok := data.miners[deal.Miner]; ok {
					data.miners[deal.Miner] = data.miners[deal.Miner] - MinerPunishmentScore
					if data.miners[deal.Miner] < 1 {
						data.miners[deal.Miner] = 1
					}
				}
			}
		}
	}
	if m.retrievals.Len() >= RetrieveParallelNum {
		log.Infof("wait for retrieval:%d", m.retrievals.Len())
		return nil
	}

	keys := m.dataRefs.Keys()
	for _, rk := range keys {
		dataObj, _ := m.dataRefs.Get(rk)
		data := dataObj.(*DataRef)

		if data.isRetrieved {
			continue
		}

		if m.retrievals.Contains(rk) {
			continue
		}

		if ok, _ := m.api.ClientHasLocal(ctx, data.rootCID); ok {
			if _, err := m.api.ClientDealSize(ctx, data.rootCID); err == nil {
				log.Infof("data has been storaged in daemon:%s", data.pieceID)
				if !data.isRetrieved {
					data.isRetrieved = true
					m.totalRetrieveCount++
				}
				continue
			}
		}

		if stored, err := m.api.StateMinerStoredAnyPiece(ctx, m.miner, []cid.Cid{data.pieceID}, types.EmptyTSK); err != nil {
			log.Debugf("failed to check miner stored piece: %s", err)
			continue
		} else if stored {
			log.Infof("data has been storaged in miner:%s", data.pieceID)
			if !data.isRetrieved {
				data.isRetrieved = true
				m.totalRetrieveCount++
			}
			continue
		}

		if m.retrievals.Len() >= RetrieveParallelNum {
			log.Infof("wait for retrieval:%d", m.retrievals.Len())
			break
		}

		if data.tryCount > 3 {
			if time.Now().Before(data.retrieveTime.Add(30 * time.Minute)) {
				continue
			}
		}

		var addrs []address.Address
		for k, v := range data.miners {
			for i := 0; i < v; i++ {
				addrs = append(addrs, k)
			}
		}

		miner := addrs[rand.Intn(len(addrs))]
		deal, err := m.api.ClientRetrieveQuery(ctx, m.minerInfo.Owner, data.rootCID, &data.pieceID, miner)
		if err != nil {
			if _, ok := data.miners[miner]; ok {
				data.miners[miner] = data.miners[miner] - MinerPunishmentScore
				if data.miners[miner] < 1 {
					data.miners[miner] = 1
				}
			}
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
		data.tryCount++
		data.retrieveTime = time.Now()
		log.Warnf("client retrieve miner:%s, data:%s", miner, data.rootCID)

		m.retrievals.Add(rk, deal)
	}
	return nil
}

func checkDealStatus(deal *api.DealInfo) (bool, bool) {
	// isDealed := (deal.State == storagemarket.StorageDealAwaitingPreCommit ||
	// 	deal.State == storagemarket.StorageDealSealing ||
	// 	deal.State == storagemarket.StorageDealActive)
	isDealed := deal.State == storagemarket.StorageDealActive
	isError := (deal.State == storagemarket.StorageDealProposalNotFound ||
		deal.State == storagemarket.StorageDealProposalRejected ||
		deal.State == storagemarket.StorageDealFailing ||
		deal.State == storagemarket.StorageDealError)
	return isDealed || isError, isDealed
}

func (m *MinerData) loadStorageData(ctx context.Context) (*lru.ARCCache, error) {
	storages, err := lru.NewARC(DealParallelNum * 2)
	if err != nil {
		return nil, err
	}

	deals, err := m.api.ClientListDeals(ctx)
	if err != nil {
		return nil, err
	}
	for _, d := range deals {
		if d.Provider == m.miner {
			dataObj, ok := m.dataRefs.Get(d.PieceCID.String())
			if ok {
				isFinish, isDealed := checkDealStatus(&d)
				if isDealed {
					data := dataObj.(*DataRef)
					data.isDealed = true
					m.totalDealCount++
				}
				if !isFinish {
					storages.Add(d.PieceCID.String(), d.ProposalCid)
				}
			}
		}
	}
	return storages, nil
}

func (m *MinerData) storageChainData(ctx context.Context) error {
	if m.deals == nil {
		lru, err := m.loadStorageData(ctx)
		if err != nil {
			return err
		}
		m.deals = lru
	}

	dealKeys := m.deals.Keys()
	for _, rk := range dealKeys {
		id, _ := m.deals.Get(rk)
		dealID := id.(cid.Cid)
		deal, err := m.api.ClientGetDealInfo(ctx, dealID)
		if err != nil {
			return err
		}
		isFinish, isDealed := checkDealStatus(deal)
		dataObj, _ := m.dataRefs.Get(rk)
		data := dataObj.(*DataRef)
		if !data.isDealed && isDealed {
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

	keys := m.dataRefs.Keys()
	for _, rk := range keys {
		dataObj, _ := m.dataRefs.Get(rk)
		data := dataObj.(*DataRef)

		// if data not found local, go to next one
		if !data.isRetrieved {
			continue
		}

		if data.isDealed {
			continue
		}

		// if miner is dealing, go to next one
		if m.deals.Contains(rk) {
			continue
		}

		if stored, err := m.api.StateMinerStoredAnyPiece(ctx, m.miner, []cid.Cid{data.pieceID}, types.EmptyTSK); err != nil {
			log.Warnf("failed to check miner stored piece: %w", err)
			continue
		} else if stored {
			log.Infof("data has been storaged:%s, error:%s", data.pieceID, err)
			if !data.isDealed {
				data.isDealed = true
				m.totalDealCount++
			}
			continue
		}

		if m.deals.Len() >= DealParallelNum {
			log.Infof("wait for deal:%d", m.deals.Len())
			break
		}

		stData := &storagemarket.DataRef{
			TransferType: storagemarket.TTGraphsync,
			Root:         data.rootCID,
		}
		params := &api.StartDealParams{
			Data:   stData,
			Wallet: address.Undef,
			Miner:  m.miner,
			/* EpochPrice:        ask.Price,
			MinBlocksDuration: uint64(ask.Expiry - ts.Height()), */
			FastRetrieval: true,
		}
		dealID, err := m.api.ClientStartDeal(ctx, params)
		if err != nil {
			log.Warnf("failed to start deal: %s", err)
			continue
		}
		log.Warnf("start deal with miner:%s deal: %s", m.miner, dealID.String())
		m.deals.Add(rk, *dealID)
	}
	return nil
}
