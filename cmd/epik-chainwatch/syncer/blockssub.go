package syncer

import (
	"context"
	"time"

	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/ipfs/go-cid"
<<<<<<< HEAD:cmd/epik-chainwatch/blockssub.go

	aapi "github.com/EpiK-Protocol/go-epik/api"
=======
>>>>>>> c1a814c23558d2bd4014355bc8c85491ff0659d4:cmd/epik-chainwatch/syncer/blockssub.go
)

func (s *Syncer) subBlocks(ctx context.Context) {
	sub, err := s.node.SyncIncomingBlocks(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	for bh := range sub {
		err := s.storeHeaders(map[cid.Cid]*types.BlockHeader{
			bh.Cid(): bh,
		}, false, time.Now())
		if err != nil {
			log.Errorf("%+v", err)
		}
	}
}
