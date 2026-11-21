// SPDX-License-Identifier: AGPL-3.0-only

package marketdata

import (
	"context"
	"time"
)

// Connector is a market data source. Connectors translate the source's protocol
// and nothing else; the core retries, validates and stores.
type Connector interface {
	Info() ConnectorInfo

	// History returns the closed candles of one series with From <= T < To,
	// oldest first. It pages through the source as needed. The candles'
	// InstrumentID is left zero for the caller to set.
	History(ctx context.Context, req HistoryRequest) ([]Candle, error)

	// Stream sends updates for subs to sink until ctx is done or the stream
	// fails. Reconnecting is the caller's job.
	Stream(ctx context.Context, subs []StreamSub, sink StreamSink) error
}

// HistoryRequest selects a range of one series by the source's own symbol.
type HistoryRequest struct {
	Symbol   string
	TF       Timeframe
	From, To int64 // open times, Unix ms UTC
}

// ConnectorInfo is a connector's static metadata.
type ConnectorInfo struct {
	ID, Name, Version string
	TermsURL          string // the source's API terms of use
	Auth              Auth
	Freshness         Freshness
	NativeTimeframes  []Timeframe // available from History
	StreamTimeframes  []Timeframe // available from Stream
	Capabilities      []Capability
	Limits            Limits
	SettingsSchema    string // JSON Schema of the connector's user settings
}

// Auth is what a source needs to serve market data.
type Auth string

// Authentication modes.
const (
	AuthNone         Auth = "none"
	AuthAPIKey       Auth = "api_key"
	AuthAPIKeySecret Auth = "api_key_secret"
	AuthOAuth        Auth = "oauth"
)

// Freshness is "realtime", "eod" or "delayed(<duration>)", such as
// "delayed(15m)".
type Freshness string

// Freshness values that need no duration.
const (
	FreshnessRealtime Freshness = "realtime"
	FreshnessEOD      Freshness = "eod"
)

// Capability is an optional feature of a source.
type Capability string

// Optional source features.
const (
	CapabilityTrades           Capability = "trades"
	CapabilityOrderbook        Capability = "orderbook"
	CapabilityExtendedHours    Capability = "extended_hours"
	CapabilityCorporateActions Capability = "corporate_actions"
	CapabilityBulkHistory      Capability = "bulk_history"
)

// Limits are a source's documented budgets. Zero means no limit.
type Limits struct {
	MaxBarsPerRequest        int
	RequestWeight            int // per RequestWeightWindow, per IP
	RequestWeightWindow      time.Duration
	MaxStreamsPerConnection  int
	ConnectionAttempts       int // per ConnectionAttemptsWindow, per IP
	ConnectionAttemptsWindow time.Duration
	MaxConnectionLifetime    time.Duration
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
