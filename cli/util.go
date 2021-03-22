package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/ipfs/go-cid"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/specs-actors/v2/actors/builtin/miner"
)

func parseTipSet(ctx context.Context, api api.FullNode, vals []string) (*types.TipSet, error) {
	var headers []*types.BlockHeader
	for _, c := range vals {
		blkc, err := cid.Decode(c)
		if err != nil {
			return nil, err
		}

		bh, err := api.ChainGetBlock(ctx, blkc)
		if err != nil {
			return nil, err
		}

		headers = append(headers, bh)
	}

	return types.NewTipSet(headers)
}

func parseFrom(cctx *cli.Context, ctx context.Context, api api.FullNode, useDef bool) (address.Address, error) {
	from := cctx.String("from")
	if from == "" {
		if !useDef {
			return address.Undef, fmt.Errorf("from not set")
		}
		return api.WalletDefaultAddress(ctx)
	}

	return address.NewFromString(from)
}

func EpochTime(curr, e abi.ChainEpoch) string {
	switch {
	case e == miner.NoExpireEpoch:
		return fmt.Sprint("never")
	case curr > e:
		return fmt.Sprintf("%d (%s ago)", e, durafmt.Parse(time.Second*time.Duration(int64(build.BlockDelaySecs)*int64(curr-e))).LimitFirstN(2))
	case curr == e:
		return fmt.Sprintf("%d (now)", e)
	case curr < e:
		return fmt.Sprintf("%d (in %s)", e, durafmt.Parse(time.Second*time.Duration(int64(build.BlockDelaySecs)*int64(e-curr))).LimitFirstN(2))
	}

	panic("math broke")
}

func LoadTipSet(ctx context.Context, cctx *cli.Context, api api.FullNode) (*types.TipSet, error) {
	tss := cctx.String("tipset")
	if tss == "" {
		return nil, nil
	}

	return ParseTipSetRef(ctx, api, tss)
}

func ParseTipSetRef(ctx context.Context, api api.FullNode, tss string) (*types.TipSet, error) {
	if tss[0] == '@' {
		if tss == "@head" {
			return api.ChainHead(ctx)
		}

		var h uint64
		if _, err := fmt.Sscanf(tss, "@%d", &h); err != nil {
			return nil, xerrors.Errorf("parsing height tipset ref: %w", err)
		}

		return api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(h), types.EmptyTSK)
	}

	cids, err := ParseTipSetString(tss)
	if err != nil {
		return nil, err
	}

	if len(cids) == 0 {
		return nil, nil
	}

	k := types.NewTipSetKey(cids...)
	ts, err := api.ChainGetTipSet(ctx, k)
	if err != nil {
		return nil, err
	}

	return ts, nil
}

func ParseTipSetString(ts string) ([]cid.Cid, error) {
	strs := strings.Split(ts, ",")

	var cids []cid.Cid
	for _, s := range strs {
		c, err := cid.Parse(strings.TrimSpace(s))
		if err != nil {
			return nil, err
		}
		cids = append(cids, c)
	}

	return cids, nil
}
