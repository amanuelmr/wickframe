// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import "context"

// Connector is a market data source. Connectors translate the source's protocol
// and nothing else; the core retries, validates and stores.
type Connector interface {
	Info() ConnectorInfo

	// Stream sends updates for subs to sink until ctx is done or the stream
	// fails. Reconnecting is the caller's job.
	Stream(ctx context.Context, subs []StreamSub, sink StreamSink) error
}

// ConnectorInfo is a connector's static metadata.
type ConnectorInfo struct {
	ID, Name string
}

// StreamSub selects one live candle series by the source's own symbol.
type StreamSub struct {
	Symbol string
	TF     Timeframe
}

// StreamSink receives a stream's updates. Calls come from one goroutine.
type StreamSink interface {
	Candle(u CandleUpdate)
}

// CandleUpdate is a forming or closed candle from a source. A closed candle
// sent again with a higher Rev is a correction.
type CandleUpdate struct {
	Sub           StreamSub
	T             int64 // open time, Unix ms UTC
	O, H, L, C, V float64
	Closed        bool
	Rev           int16
}
