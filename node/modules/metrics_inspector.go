// Copyright 2021 jasonhuang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modules

import (
	"context"
	"math/big"
	"time"

	"github.com/EpiK-Protocol/go-epik/api"
	"github.com/EpiK-Protocol/go-epik/build"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	sealing "github.com/EpiK-Protocol/go-epik/extern/storage-sealing"
	"github.com/EpiK-Protocol/go-epik/metrics"
	"github.com/EpiK-Protocol/go-epik/node/modules/helpers"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	metrics2 "github.com/libp2p/go-libp2p-core/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/fx"
)

func RunMinerSysMetrics(mctx helpers.MetricsCtx, lc fx.Lifecycle, reporter metrics2.Reporter) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	go metrics.RunSysInspector(ctx, reporter, 5*time.Second, "miner")
}

func RunChainSysMetrics(mctx helpers.MetricsCtx, lc fx.Lifecycle, reporter metrics2.Reporter) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	go metrics.RunSysInspector(ctx, reporter, 5*time.Second, "chain")
}

func RunMinerMetrics(mctx helpers.MetricsCtx, lc fx.Lifecycle, node api.FullNode, minerapi api.StorageMiner, reporter metrics2.Reporter) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	stop := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				ticker := time.NewTicker(time.Duration(build.BlockDelaySecs) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-stop:
						return
					case <-ticker.C:
						miner, err := minerapi.ActorAddress(ctx)
						if err != nil {
							log.Warnf("failed to get miner: %w", err)
							continue
						}

						if err := recordCoinbase(ctx, node, miner); err != nil {
							log.Warnf("failed to record coinbase: %w", err)
							continue
						}

						if err := recordMinerPower(ctx, node, miner); err != nil {
							log.Warnf("failed to record miner power: %w", err)
							continue
						}

						if err := recordMinerSector(ctx, minerapi, miner); err != nil {
							log.Warnf("failed to record miner sector: %w", err)
							continue
						}
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			close(stop)
			return nil
		},
	})
}

func recordCoinbase(ctx context.Context, node api.FullNode, miner address.Address) error {
	b2f := func(amt abi.TokenAmount) float64 {
		f := 0.0
		r := new(big.Rat).SetFrac(amt.Int, big.NewInt(int64(build.EpkPrecision)))
		if r.Sign() != 0 {
			f, _ = r.Float64()
		}
		return f
	}

	mi, err := node.StateMinerInfo(ctx, miner, types.EmptyTSK)
	if err != nil {
		return err
	}
	tagsT := []tag.Mutator{
		tag.Insert(metrics.Coinbase, mi.Coinbase.String()),
		tag.Insert(metrics.Type, "total"),
	}
	tagsA := []tag.Mutator{
		tag.Insert(metrics.Coinbase, mi.Coinbase.String()),
		tag.Insert(metrics.Type, "available"),
	}

	ci, err := node.StateCoinbase(ctx, mi.Coinbase, types.EmptyTSK)
	if err != nil {
		return err
	}

	stats.RecordWithTags(ctx, tagsT, metrics.CoinbaseBalance.M(b2f(ci.Total)))
	stats.RecordWithTags(ctx, tagsA, metrics.CoinbaseBalance.M(b2f(ci.Vested)))
	return nil
}

func recordMinerPower(ctx context.Context, node api.FullNode, miner address.Address) error {
	tagsPR := []tag.Mutator{
		tag.Insert(metrics.MinerID, miner.String()),
		tag.Insert(metrics.Type, "raw"),
	}

	tagsPQ := []tag.Mutator{
		tag.Insert(metrics.MinerID, miner.String()),
		tag.Insert(metrics.Type, "quality"),
	}

	p, err := node.StateMinerPower(ctx, miner, types.EmptyTSK)
	if err != nil {
		return err
	}

	stats.RecordWithTags(ctx, tagsPR, metrics.MinerPower.M(p.MinerPower.RawBytePower.Int64()))
	stats.RecordWithTags(ctx, tagsPQ, metrics.MinerPower.M(p.MinerPower.QualityAdjPower.Int64()))
	return nil
}

func recordMinerSector(ctx context.Context, minerapi api.StorageMiner, miner address.Address) error {
	summary, err := minerapi.SectorsSummary(ctx)
	if err != nil {
		return err
	}

	buckets := make(map[sealing.SectorState]int)
	var total int
	for s, c := range summary {
		buckets[sealing.SectorState(s)] = c
		total += c
	}
	buckets["Total"] = total

	for state, count := range buckets {
		tags := []tag.Mutator{
			tag.Insert(metrics.MinerID, miner.String()),
			tag.Insert(metrics.Type, string(state)),
		}
		stats.RecordWithTags(ctx, tags, metrics.MinerSectorCount.M(int64(count)))
	}
	return nil
}
