package metrics

import (
	"context"
	"math"
	"time"

	logging "github.com/ipfs/go-log/v2"
	p2pmetrics "github.com/libp2p/go-libp2p-core/metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

var log = logging.Logger("node-metrics")

func RunSysInspector(ctx context.Context, reporter p2pmetrics.Reporter, interval time.Duration, done <-chan struct{}) {

	tagsIn := []tag.Mutator{tag.Insert(Type, "in")}
	tagsOut := []tag.Mutator{tag.Insert(Type, "out")}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	cpuCh := make(chan int64)
	go getCpuUsage(ctx, interval, cpuCh, done)

	for {
		select {
		case <-done:
			return
		case usage := <-cpuCh:
			stats.Record(ctx, SysCpuUsed.M(usage))
		case <-ticker.C:
			totals := reporter.GetBandwidthTotals()
			stats.RecordWithTags(ctx, tagsIn, BandwidthTotal.M(totals.TotalIn))
			stats.RecordWithTags(ctx, tagsIn, BandwidthRate.M(totals.RateIn))
			stats.RecordWithTags(ctx, tagsOut, BandwidthTotal.M(totals.TotalOut))
			stats.RecordWithTags(ctx, tagsOut, BandwidthRate.M(totals.RateOut))

			stats.Record(ctx, SysMemUsed.M(getMemUsage()))
			stats.Record(ctx, SysDiskUsed.M(getDiskUsage()))
		}
	}
}

func getCpuUsage(ctx context.Context, interval time.Duration, out chan<- int64, done <-chan struct{}) {
	if interval < time.Second {
		log.Errorf("interval too short: %d", interval)
		return
	}
	for {
		select {
		case <-done:
			return
		default:
			used, err := cpu.PercentWithContext(ctx, interval, false)
			if err != nil {
				log.Errorf("cpu inspecting failed: %w", err)
				continue
			}
			out <- int64(math.Round(used[0]))
		}
	}
}

func getMemUsage() int64 {
	stat, err := mem.VirtualMemory()
	if err != nil {
		log.Errorf("mem inspecting failed: %w", err)
		return 0
	}

	return int64(math.Round(stat.UsedPercent))
}

func getDiskUsage() int64 {
	stat, err := disk.Usage("/")
	if err != nil {
		log.Errorf("disk inspecting failed: %w", err)
		return 0
	}
	return int64(math.Round(stat.UsedPercent))
}
