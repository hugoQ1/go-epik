package genesis

import (
	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
)

type InitDatas struct {
	Expert          address.Address
	ExpertOwner     address.Address
	PresealPieceCID cid.Cid
	PresealRootCID  cid.Cid // fake genesis root CID
}
