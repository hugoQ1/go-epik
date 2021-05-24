package store

import (
	"context"
	"time"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/ipfs/go-cid"
	"go.opencensus.io/trace"
)

type heaviestNotify struct {
	old *types.TipSet
	new *types.TipSet
}

func (cs *ChainStore) newTakeHeaviestTipSet(ctx context.Context, ts *types.TipSet) error {
	_, span := trace.StartSpan(ctx, "newTakeHeaviestTipSet")
	defer span.End()

	if len(cs.notifyCh) > 5 {
		log.Warnf("slow notify %d", len(cs.notifyCh))
	}

	cs.notifyCh <- &heaviestNotify{
		old: cs.heaviest,
		new: ts,
	}

	span.AddAttributes(trace.BoolAttribute("newHead", true))

	log.Infof("New heaviest tipset! %s (height=%d)", ts.Cids(), ts.Height())
	cs.heaviest = ts
	if err := cs.writeHead(ts); err != nil {
		log.Errorf("failed to write chain head: %s", err)
	}
	return nil
}

func (cs *ChainStore) headCompressor(ctx context.Context) chan<- *heaviestNotify {

	out := make(chan *heaviestNotify, 32)

	pubNotify := func(notify *heaviestNotify) {
		if notify.old != nil { // buf
			if len(cs.reorgCh) > 0 {
				log.Warnf("Reorg channel running behind, %d reorgs buffered", len(cs.reorgCh))
			}
			cs.reorgCh <- reorg{
				old: notify.old,
				new: notify.new,
			}
		} else {
			log.Warnf("no heaviest tipset found, using %s", notify.new.Cids())
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
			prevNotify *heaviestNotify
			lastRecv   time.Time
			prevCids   map[cid.Cid]struct{}
		)
		for {
			select {
			case notify := <-out:
				lastRecv = time.Now()

				if prevNotify == nil {
					prevNotify = notify
					prevCids = convertToMap(notify.new.Cids())
					break
				}

				newCids := convertToMap(notify.new.Cids())

				containsPrev := false
				if prevNotify.new.Height() == notify.new.Height() {
					containsPrev = contains(newCids, prevCids)
				}
				if containsPrev && notify.old == prevNotify.new {
					prevNotify = &heaviestNotify{
						old: prevNotify.old,
						new: notify.new,
					}
					if prevNotify.old != nil {
						log.Infof("merge heaviest notifications! old: %s (height=%d), new: %s (height=%d)",
							prevNotify.old.Cids(), prevNotify.old.Height(),
							prevNotify.new.Cids(), prevNotify.new.Height(),
						)
					} else {
						log.Infof("merge heaviest notifications! %s (height=%d)", prevNotify.new.Cids(), prevNotify.new.Height())
					}
				} else {
					pubNotify(prevNotify)
					prevNotify = notify
				}

				prevCids = newCids

			case <-ticker1.C:
				if prevNotify != nil && time.Since(lastRecv) >= time.Second {
					pubNotify(prevNotify)
					prevNotify = nil
					prevCids = nil
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}
