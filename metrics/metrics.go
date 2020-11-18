package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	rpcmetrics "github.com/filecoin-project/go-jsonrpc/metrics"
	_ "github.com/influxdata/influxdb1-client"
)

// Global Tags
var (
	Version, _      = tag.NewKey("version")
	Commit, _       = tag.NewKey("commit")
	PeerID, _       = tag.NewKey("peer_id")
	FailureType, _  = tag.NewKey("failure_type")
	MessageFrom, _  = tag.NewKey("message_from")
	MessageTo, _    = tag.NewKey("message_to")
	MessageNonce, _ = tag.NewKey("message_nonce")
	ReceivedFrom, _ = tag.NewKey("received_from")
)

// Measures
var (
	EpikInfo                            = stats.Int64("info", "Arbitrary counter to tag epik info to", stats.UnitDimensionless)
	ChainNodeHeight                     = stats.Int64("chain/node_height", "Current Height of the node", stats.UnitDimensionless)
	ChainNodeWorkerHeight               = stats.Int64("chain/node_worker_height", "Current Height of workers on the node", stats.UnitDimensionless)
	MessageReceived                     = stats.Int64("message/received", "Counter for total received messages", stats.UnitDimensionless)
	MessageReceivedSize                 = stats.Int64("message/received_size", "Counter for total bytes of received messages", stats.UnitBytes)
	MessageValidationFailure            = stats.Int64("message/failure", "Counter for message validation failures", stats.UnitDimensionless)
	MessageValidationSuccess            = stats.Int64("message/success", "Counter for message validation successes", stats.UnitDimensionless)
	BlockReceived                       = stats.Int64("block/received", "Counter for total received blocks", stats.UnitDimensionless)
	BlockReceivedSize                   = stats.Int64("block/received_size", "Counter for total bytes of received blocks", stats.UnitBytes)
	BlockValidationFailure              = stats.Int64("block/failure", "Counter for block validation failures", stats.UnitDimensionless)
	BlockValidationSuccess              = stats.Int64("block/success", "Counter for block validation successes", stats.UnitDimensionless)
	BlockValidationDurationMilliseconds = stats.Float64("block/validation_ms", "Duration for Block Validation in ms", stats.UnitMilliseconds)
	PeerCount                           = stats.Int64("peer/count", "Current number of EPK peers", stats.UnitDimensionless)
	// bandwidth
	BandwidthTotalIn  = stats.Int64("bandwidth/total_in", "Counter for total traffic in", stats.UnitBytes)
	BandwidthTotalOut = stats.Int64("bandwidth/total_out", "Counter for total traffic out", stats.UnitBytes)
	BandwidthRateIn   = stats.Float64("bandwidth/rate_in", "Counter for bytes received per second", stats.UnitBytes)
	BandwidthRateOut  = stats.Float64("bandwidth/rate_out", "Counter for bytes sent per second", stats.UnitBytes)
	// retrieval
	RetrievalReceived = stats.Int64("retrieval/received", "Counter for total received retrieval", stats.UnitDimensionless)
	RetrievalAccepted = stats.Int64("retrieval/accepted", "Counter for total accepted retrieval", stats.UnitDimensionless)
	// sync
	SyncResponse        = stats.Int64("sync/response", "Counter for total sync response", stats.UnitDimensionless)
	SyncResponseFailure = stats.Int64("sync/response_failure", "Counter for failed sync response", stats.UnitDimensionless)
	SyncSent            = stats.Int64("sync/sent", "Counter for total sent bytes of sync", stats.UnitBytes)
	SyncRequest         = stats.Int64("sync/request", "Counter for total sync request", stats.UnitDimensionless)
	SyncRequestFailure  = stats.Int64("sync/request_failure", "Counter for failed sync request", stats.UnitDimensionless)
	SyncReceived        = stats.Int64("sync/received", "Counter for total received bytes of sync", stats.UnitBytes)
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
	ChainNodeWorkerHeightView = &view.View{
		Measure:     ChainNodeWorkerHeight,
		Aggregation: view.LastValue(),
	}
	BlockReceivedView = &view.View{
		Measure:     BlockReceived,
		Aggregation: view.Count(),
	}
	BlockReceivedSizeView = &view.View{
		Measure:     BlockReceivedSize,
		Aggregation: view.Sum(),
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
		Aggregation: view.Sum(),
	}
	MessageReceivedView = &view.View{
		Measure:     MessageReceived,
		Aggregation: view.Count(),
	}
	MessageReceivedSizeView = &view.View{
		Measure:     MessageReceivedSize,
		Aggregation: view.Sum(),
	}
	MessageValidationFailureView = &view.View{
		Measure:     MessageValidationFailure,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{FailureType},
	}
	MessageValidationSuccessView = &view.View{
		Measure:     MessageValidationSuccess,
		Aggregation: view.Count(),
	}
	PeerCountView = &view.View{
		Measure:     PeerCount,
		Aggregation: view.LastValue(),
	}
	BandwidthTotalInView = &view.View{
		Measure:     BandwidthTotalIn,
		Aggregation: view.LastValue(),
	}
	BandwidthTotalOutView = &view.View{
		Measure:     BandwidthTotalOut,
		Aggregation: view.LastValue(),
	}
	BandwidthRateInView = &view.View{
		Measure:     BandwidthRateIn,
		Aggregation: view.LastValue(),
	}
	BandwidthRateOutView = &view.View{
		Measure:     BandwidthRateOut,
		Aggregation: view.LastValue(),
	}
	RetrievalReceivedView = &view.View{
		Measure:     RetrievalReceived,
		Aggregation: view.Count(),
	}
	RetrievalAcceptedView = &view.View{
		Measure:     RetrievalAccepted,
		Aggregation: view.Count(),
	}
	SyncRequestView = &view.View{
		Measure:     SyncRequest,
		Aggregation: view.Count(),
	}
	SyncRequestFailureView = &view.View{
		Measure:     SyncRequestFailure,
		Aggregation: view.Count(),
	}
	SyncReceivedView = &view.View{
		Measure:     SyncReceived,
		Aggregation: view.Sum(),
	}
	SyncResponseView = &view.View{
		Measure:     SyncResponse,
		Aggregation: view.Count(),
	}
	SyncResponseFailureView = &view.View{
		Measure:     SyncResponseFailure,
		Aggregation: view.Count(),
	}
	SyncSentView = &view.View{
		Measure:     SyncSent,
		Aggregation: view.Sum(),
	}
)

// DefaultViews is an array of OpenCensus views for metric gathering purposes
var DefaultViews = append([]*view.View{
	InfoView,
	ChainNodeHeightView,
	ChainNodeWorkerHeightView,
	BlockReceivedView,
	BlockReceivedSizeView,
	BlockValidationFailureView,
	BlockValidationSuccessView,
	BlockValidationDurationView,
	MessageReceivedView,
	MessageReceivedSizeView,
	MessageValidationFailureView,
	MessageValidationSuccessView,
	PeerCountView,
	BandwidthTotalInView,
	BandwidthTotalOutView,
	BandwidthRateInView,
	BandwidthRateOutView,
	SyncRequestView,
	SyncRequestFailureView,
	SyncReceivedView,
	SyncResponseView,
	SyncResponseFailureView,
	SyncSentView,
}, rpcmetrics.DefaultViews...)
