package store

import (
	"context"
	"time"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/ipfs/go-cid"
	"go.opencensus.io/trace"
)

func (cs *ChainStore) compressedTakeHeaviestTipSet(ctx context.Context, ts *types.TipSet) error {
	if len(cs.compressingCh) > 2 {
		log.Warnf("slow head compressing %d", len(cs.compressingCh))
	}
	if cs.heaviest == nil {
		cs.heaviest = ts
	} else {
		cs.compressingCh <- ts
	}
	return nil
}

func (cs *ChainStore) headCompressor(ctx context.Context) chan<- *types.TipSet {

	out := make(chan *types.TipSet, 32)

	pushCompressedTs := func(ts *types.TipSet) {
		_, span := trace.StartSpan(ctx, "pushCompressedTs")
		defer span.End()

		if cs.heaviest != nil { // buf
			if len(cs.reorgCh) > 0 {
				log.Warnf("Reorg channel running behind, %d reorgs buffered", len(cs.reorgCh))
			}
			cs.reorgCh <- reorg{
				old: cs.heaviest,
				new: ts,
			}
		} else {
			log.Warnf("no heaviest tipset found, using %s", ts.Cids())
		}

		span.AddAttributes(trace.BoolAttribute("newHead", true))

		log.Infof("New heaviest tipset! %s (height=%d)", ts.Cids(), ts.Height())
		cs.heaviest = ts

		if err := cs.writeHead(ts); err != nil {
			log.Errorf("failed to write chain head: %s", err)
		}
	}

	convertToMap := func(ids []cid.Cid) map[cid.Cid]struct{} {
		out := make(map[cid.Cid]struct{})
		for _, id := range ids {
			out[id] = struct{}{}
		}
		return out
	}

	// return a.contains(b)
	contains := func(a, b map[cid.Cid]struct{}) bool {
		for id := range b {
			if _, ok := a[id]; !ok {
				return false
			}
		}
		return true
	}

	cs.wg.Add(1)
	go func() {
		defer cs.wg.Done()
		defer log.Warn("headCompressor quit")

		ticker1 := time.NewTicker(time.Second)
		defer ticker1.Stop()

		var (
			prevBest *types.TipSet
			lastRecv time.Time
			prevCids map[cid.Cid]struct{}
		)
		for {
			select {
			case r := <-out:
				lastRecv = time.Now()

				if prevBest == nil {
					prevBest = r
					prevCids = convertToMap(r.Cids())
					break
				}

				newCids := convertToMap(r.Cids())

				containsPrev := false
				if prevBest.Height() == r.Height() {
					containsPrev = contains(newCids, prevCids)
				}
				if !containsPrev {
					pushCompressedTs(prevBest)
				} else {
					log.Infof("Compress heaviest tipset! %s (height=%d)", r.Cids(), r.Height())
				}

				prevBest = r
				prevCids = newCids

			case <-ticker1.C:
				if prevBest != nil && time.Since(lastRecv) >= time.Second {
					pushCompressedTs(prevBest)
					prevBest = nil
					prevCids = nil
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
