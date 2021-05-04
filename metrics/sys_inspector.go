package metrics

import (
	"context"
	"math"
	"time"

	p2pmetrics "github.com/libp2p/go-libp2p-core/metrics"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func RunSysInspector(ctx context.Context, reporter p2pmetrics.Reporter, interval time.Duration, nodeType string) {

	tagsIn := []tag.Mutator{tag.Insert(Type, "in"), tag.Insert(NodeType, nodeType)}
	tagsOut := []tag.Mutator{tag.Insert(Type, "out"), tag.Insert(NodeType, nodeType)}
	tagsCom := []tag.Mutator{tag.Insert(NodeType, nodeType)}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	cpuCh := make(chan int64)
	go getCpuUsage(ctx, interval, cpuCh)

	for {
		select {
		case <-ctx.Done():
			return
		case usage := <-cpuCh:
			stats.RecordWithTags(ctx, tagsCom, SysCpuUsed.M(usage))
		case <-ticker.C:
			totals := reporter.GetBandwidthTotals()
			stats.RecordWithTags(ctx, tagsIn, BandwidthTotal.M(totals.TotalIn))
			stats.RecordWithTags(ctx, tagsIn, BandwidthRate.M(totals.RateIn))
			stats.RecordWithTags(ctx, tagsOut, BandwidthTotal.M(totals.TotalOut))
			stats.RecordWithTags(ctx, tagsOut, BandwidthRate.M(totals.RateOut))

			stats.RecordWithTags(ctx, tagsCom, SysMemUsed.M(getMemUsage()))
			stats.RecordWithTags(ctx, tagsCom, SysDiskUsed.M(getDiskUsage()))
		}
	}
}

func getCpuUsage(ctx context.Context, interval time.Duration, out chan<- int64) {
	if interval < time.Second {
		log.Errorf("interval too short: %d", interval)
		return
	}
	for {
		select {
		case <-ctx.Done():
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
