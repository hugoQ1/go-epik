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
	"github.com/EpiK-Protocol/go-epik/metrics"
	"github.com/EpiK-Protocol/go-epik/node/modules/dtypes"
	"github.com/EpiK-Protocol/go-epik/node/modules/helpers"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	metrics2 "github.com/libp2p/go-libp2p-core/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.uber.org/fx"
)

func RunSysMetrics(mctx helpers.MetricsCtx, lc fx.Lifecycle, reporter metrics2.Reporter) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	stop := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go metrics.RunSysInspector(ctx, reporter, 5*time.Second, stop)
			return nil
		},
		OnStop: func(context.Context) error {
			close(stop)
			return nil
		},
	})
}

func RunMinerMetrics(mctx helpers.MetricsCtx, lc fx.Lifecycle, node api.FullNode, minerAddress dtypes.MinerAddress, reporter metrics2.Reporter) {
	ctx := helpers.LifecycleCtx(mctx, lc)
	stop := make(chan struct{})
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				b2f := func(amt abi.TokenAmount) float64 {
					f := 0.0
					r := new(big.Rat).SetFrac(amt.Int, big.NewInt(int64(build.EpkPrecision)))
					if r.Sign() != 0 {
						f, _ = r.Float64()
					}
					return f
				}

				tagsT := []tag.Mutator{
					tag.Insert(metrics.MinerID, address.Address(minerAddress).String()),
					tag.Insert(metrics.Type, "total"),
				}
				tagsA := []tag.Mutator{
					tag.Insert(metrics.MinerID, address.Address(minerAddress).String()),
					tag.Insert(metrics.Type, "available"),
				}

				ticker := time.NewTicker(time.Duration(build.BlockDelaySecs) * time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-stop:
						return
					case <-ticker.C:
						head, err := node.ChainHead(ctx)
						if err != nil {
							log.Warnf("failed to get head: %w", err)
							continue
						}
						ava, err := node.StateMinerAvailableBalance(ctx, address.Address(minerAddress), head.Key())
						if err != nil {
							log.Warnf("failed to get available balance: %w", err)
							continue
						}
						act, err := node.StateGetActor(ctx, address.Address(minerAddress), head.Key())
						if err != nil {
							log.Warnf("failed to get total balance: %w", err)
							continue
						}
						stats.RecordWithTags(ctx, tagsT, metrics.MinerBalance.M(b2f(act.Balance)))
						stats.RecordWithTags(ctx, tagsA, metrics.MinerBalance.M(b2f(ava)))
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
