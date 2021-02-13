package metrics

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	rpcmetrics "github.com/filecoin-project/go-jsonrpc/metrics"
	_ "github.com/influxdata/influxdb1-client"
)

// Distribution
var defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)

// Global Tags
var (
	Version, _      = tag.NewKey("version")
	Commit, _       = tag.NewKey("commit")
	PeerID, _       = tag.NewKey("peer_id")
	MinerID, _      = tag.NewKey("miner_id")
	FailureType, _  = tag.NewKey("failure_type")
	Local, _        = tag.NewKey("local")
	MessageFrom, _  = tag.NewKey("message_from")
	MessageTo, _    = tag.NewKey("message_to")
	MessageNonce, _ = tag.NewKey("message_nonce")
	ReceivedFrom, _ = tag.NewKey("received_from")
	Endpoint, _     = tag.NewKey("endpoint")
	APIInterface, _ = tag.NewKey("api") // to distinguish between gateway api and full node api endpoint calls

	Type, _ = tag.NewKey("type")
)

// Measures
var (
	EpikInfo                            = stats.Int64("info", "Arbitrary counter to tag epik info to", stats.UnitDimensionless)
	ChainNodeHeight                     = stats.Int64("chain/node_height", "Current Height of the node", stats.UnitDimensionless)
	ChainNodeHeightExpected             = stats.Int64("chain/node_height_expected", "Expected Height of the node", stats.UnitDimensionless)
	ChainNodeWorkerHeight               = stats.Int64("chain/node_worker_height", "Current Height of workers on the node", stats.UnitDimensionless)
	MessagePublished                    = stats.Int64("message/published", "Counter for total locally published messages", stats.UnitDimensionless)
	MessageReceived                     = stats.Int64("message/received", "Counter for total received messages", stats.UnitDimensionless)
	MessageValidationFailure            = stats.Int64("message/failure", "Counter for message validation failures", stats.UnitDimensionless)
	MessageValidationSuccess            = stats.Int64("message/success", "Counter for message validation successes", stats.UnitDimensionless)
	BlockPublished                      = stats.Int64("block/published", "Counter for total locally published blocks", stats.UnitDimensionless)
	BlockReceived                       = stats.Int64("block/received", "Counter for total received blocks", stats.UnitDimensionless)
	BlockValidationFailure              = stats.Int64("block/failure", "Counter for block validation failures", stats.UnitDimensionless)
	BlockValidationSuccess              = stats.Int64("block/success", "Counter for block validation successes", stats.UnitDimensionless)
	BlockValidationDurationMilliseconds = stats.Float64("block/validation_ms", "Duration for Block Validation in ms", stats.UnitMilliseconds)
	BlockDelay                          = stats.Int64("block/delay", "Delay of accepted blocks, where delay is >5s", stats.UnitMilliseconds)
	PeerCount                           = stats.Int64("peer/count", "Current number of EPK peers", stats.UnitDimensionless)
	PubsubPublishMessage                = stats.Int64("pubsub/published", "Counter for total published messages", stats.UnitDimensionless)
	PubsubDeliverMessage                = stats.Int64("pubsub/delivered", "Counter for total delivered messages", stats.UnitDimensionless)
	PubsubRejectMessage                 = stats.Int64("pubsub/rejected", "Counter for total rejected messages", stats.UnitDimensionless)
	PubsubDuplicateMessage              = stats.Int64("pubsub/duplicate", "Counter for total duplicate messages", stats.UnitDimensionless)
	PubsubRecvRPC                       = stats.Int64("pubsub/recv_rpc", "Counter for total received RPCs", stats.UnitDimensionless)
	PubsubSendRPC                       = stats.Int64("pubsub/send_rpc", "Counter for total sent RPCs", stats.UnitDimensionless)
	PubsubDropRPC                       = stats.Int64("pubsub/drop_rpc", "Counter for total dropped RPCs", stats.UnitDimensionless)
	APIRequestDuration                  = stats.Float64("api/request_duration_ms", "Duration of API requests", stats.UnitMilliseconds)
	VMFlushCopyDuration                 = stats.Float64("vm/flush_copy_ms", "Time spent in VM Flush Copy", stats.UnitMilliseconds)
	VMFlushCopyCount                    = stats.Int64("vm/flush_copy_count", "Number of copied objects", stats.UnitDimensionless)

	// traffic
	MessageReceivedBytes = stats.Int64("message/received_bytes", "Counter for total bytes of received messages", stats.UnitBytes)
	BlockReceivedBytes   = stats.Int64("block/received_bytes", "Counter for total bytes of received blocks", stats.UnitBytes)
	BandwidthTotal       = stats.Int64("bandwidth/total", "Counter for total traffic bytes", stats.UnitBytes)
	BandwidthRate        = stats.Float64("bandwidth/rate", "Counter for bytes per second", stats.UnitBytes)
	// transfer
	ServeTransferBytes  = stats.Int64("serve/transfer_bytes", "Counter for total sent bytes", stats.UnitBytes)
	ServeTransferAccept = stats.Int64("serve/transfer_accept", "Counter for total accepted requests", stats.UnitDimensionless)
	ServeTransferResult = stats.Int64("serve/transfer_result", "Counter for process results", stats.UnitDimensionless)
	// sync
	ServeSyncSuccess = stats.Int64("serve/sync_success", "Counter for successes", stats.UnitDimensionless)
	ServeSyncFailure = stats.Int64("serve/sync_failure", "Counter for failures", stats.UnitDimensionless)
	ServeSyncBytes   = stats.Int64("serve/sync_bytes", "Counter for total sent bytes", stats.UnitBytes)
	// EPK
	MinerBalance = stats.Float64("miner/balance", "Counter for miner balance in EPK", stats.UnitDimensionless)
	// Sys
	SysCpuUsed  = stats.Int64("sys/cpu_used", "Counter for used percentage of cpu used", stats.UnitDimensionless)
	SysMemUsed  = stats.Int64("sys/mem_used", "Counter for used percentage of ram used", stats.UnitDimensionless)
	SysDiskUsed = stats.Int64("sys/disk_used", "Counter for used percentage of disk", stats.UnitDimensionless)
)

var (
	InfoView = &view.View{
		Name:        "info",
		Description: "epik node information",
		Measure:     EpikInfo,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{Version, Commit},
	}
	ChainNodeHeightView = &view.View{
		Measure:     ChainNodeHeight,
		Aggregation: view.LastValue(),
	}
	ChainNodeHeightExpectedView = &view.View{
		Measure:     ChainNodeHeightExpected,
		Aggregation: view.LastValue(),
	}
	ChainNodeWorkerHeightView = &view.View{
		Measure:     ChainNodeWorkerHeight,
		Aggregation: view.LastValue(),
	}
	BlockReceivedView = &view.View{
		Measure:     BlockReceived,
		Aggregation: view.Count(),
	}
	BlockValidationFailureView = &view.View{
		Measure:     BlockValidationFailure,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{FailureType},
	}
	BlockValidationSuccessView = &view.View{
		Measure:     BlockValidationSuccess,
		Aggregation: view.Count(),
	}
	BlockValidationDurationView = &view.View{
		Measure:     BlockValidationDurationMilliseconds,
		Aggregation: defaultMillisecondsDistribution,
	}
	BlockDelayView = &view.View{
		Measure: BlockDelay,
		TagKeys: []tag.Key{MinerID},
		Aggregation: func() *view.Aggregation {
			var bounds []float64
			for i := 5; i < 29; i++ { // 5-29s, step 1s
				bounds = append(bounds, float64(i*1000))
			}
			for i := 30; i < 60; i += 2 { // 30-58s, step 2s
				bounds = append(bounds, float64(i*1000))
			}
			for i := 60; i <= 300; i += 10 { // 60-300s, step 10s
				bounds = append(bounds, float64(i*1000))
			}
			bounds = append(bounds, 600*1000) // final cutoff at 10m
			return view.Distribution(bounds...)
		}(),
	}
	MessagePublishedView = &view.View{
		Measure:     MessagePublished,
		Aggregation: view.Count(),
	}
	MessageReceivedView = &view.View{
		Measure:     MessageReceived,
		Aggregation: view.Count(),
	}
	MessageValidationFailureView = &view.View{
		Measure:     MessageValidationFailure,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{FailureType, Local},
	}
	MessageValidationSuccessView = &view.View{
		Measure:     MessageValidationSuccess,
		Aggregation: view.Count(),
	}
	PeerCountView = &view.View{
		Measure:     PeerCount,
		Aggregation: view.LastValue(),
	}
	PubsubPublishMessageView = &view.View{
		Measure:     PubsubPublishMessage,
		Aggregation: view.Count(),
	}
	PubsubDeliverMessageView = &view.View{
		Measure:     PubsubDeliverMessage,
		Aggregation: view.Count(),
	}
	PubsubRejectMessageView = &view.View{
		Measure:     PubsubRejectMessage,
		Aggregation: view.Count(),
	}
	PubsubDuplicateMessageView = &view.View{
		Measure:     PubsubDuplicateMessage,
		Aggregation: view.Count(),
	}
	PubsubRecvRPCView = &view.View{
		Measure:     PubsubRecvRPC,
		Aggregation: view.Count(),
	}
	PubsubSendRPCView = &view.View{
		Measure:     PubsubSendRPC,
		Aggregation: view.Count(),
	}
	PubsubDropRPCView = &view.View{
		Measure:     PubsubDropRPC,
		Aggregation: view.Count(),
	}
	APIRequestDurationView = &view.View{
		Measure:     APIRequestDuration,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{APIInterface, Endpoint},
	}
	VMFlushCopyDurationView = &view.View{
		Measure:     VMFlushCopyDuration,
		Aggregation: view.Sum(),
	}
	VMFlushCopyCountView = &view.View{
		Measure:     VMFlushCopyCount,
		Aggregation: view.Sum(),
	}

	MessageReceivedBytesView = &view.View{
		Measure:     MessageReceivedBytes,
		Aggregation: view.Sum(),
	}
	BlockReceivedBytesView = &view.View{
		Measure:     BlockReceivedBytes,
		Aggregation: view.Sum(),
	}
	BandwidthTotalView = &view.View{
		Measure:     BandwidthTotal,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{Type},
	}
	BandwidthRateView = &view.View{
		Measure:     BandwidthRate,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{Type},
	}
	ServeTransferBytesView = &view.View{
		Measure:     ServeTransferBytes,
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{Type},
	}
	ServeTransferAcceptView = &view.View{
		Measure:     ServeTransferAccept,
		Aggregation: view.Count(),
	}
	ServeTransferResultView = &view.View{
		Measure:     ServeTransferResult,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Type},
	}
	ServeSyncSuccessView = &view.View{
		Measure:     ServeSyncSuccess,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{Type},
	}
	ServeSyncFailureView = &view.View{
		Measure:     ServeSyncFailure,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{FailureType},
	}
	ServeSyncBytesView = &view.View{
		Measure:     ServeSyncBytes,
		Aggregation: view.Sum(),
		TagKeys:     []tag.Key{Type},
	}
	MinerBalanceView = &view.View{
		Measure:     MinerBalance,
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{Type, MinerID},
	}
	SysCpuUsedView = &view.View{
		Measure:     SysCpuUsed,
		Aggregation: view.LastValue(),
	}
	SysMemUsedView = &view.View{
		Measure:     SysMemUsed,
		Aggregation: view.LastValue(),
	}
	SysDiskUsedView = &view.View{
		Measure:     SysDiskUsed,
		Aggregation: view.LastValue(),
	}
)

// DefaultViews is an array of OpenCensus views for metric gathering purposes
var DefaultViews = append([]*view.View{
	InfoView,
	ChainNodeHeightView,
	ChainNodeHeightExpectedView,
	ChainNodeWorkerHeightView,
	BlockReceivedView,
	BlockValidationFailureView,
	BlockValidationSuccessView,
	BlockValidationDurationView,
	BlockDelayView,
	MessagePublishedView,
	MessageReceivedView,
	MessageValidationFailureView,
	MessageValidationSuccessView,
	PeerCountView,
	PubsubPublishMessageView,
	PubsubDeliverMessageView,
	PubsubRejectMessageView,
	PubsubDuplicateMessageView,
	PubsubRecvRPCView,
	PubsubSendRPCView,
	PubsubDropRPCView,
	APIRequestDurationView,
	VMFlushCopyCountView,
	VMFlushCopyDurationView,
},
	rpcmetrics.DefaultViews...)

var EpikViews = []*view.View{
	MessageReceivedBytesView,
	BlockReceivedBytesView,
	ServeSyncSuccessView,
	ServeSyncFailureView,
	ServeSyncBytesView,
}

var MinerViews = []*view.View{
	BandwidthTotalView,
	BandwidthRateView,
	ServeTransferBytesView,
	ServeTransferAcceptView,
	ServeTransferResultView,
	MinerBalanceView,
	SysCpuUsedView,
	SysMemUsedView,
	SysDiskUsedView,
}

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

// Timer is a function stopwatch, calling it starts the timer,
// calling the returned function will record the duration.
func Timer(ctx context.Context, m *stats.Float64Measure) func() {
	start := time.Now()
	return func() {
		stats.Record(ctx, m.M(SinceInMilliseconds(start)))
	}
}
