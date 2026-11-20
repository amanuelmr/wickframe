// SPDX-License-Identifier: AGPL-3.0-only

package httpapi

import (
	"context"
	"log/slog"
	"math"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/amanuelmr/wickframe/internal/marketdata"
	"github.com/amanuelmr/wickframe/internal/platform/logx"
)

type candlesInput struct {
	Instrument string `query:"instrument" required:"true" doc:"Instrument key, such as binance:BTCUSDT." example:"fake:DEMO"`
	TF         string `query:"tf" required:"true" enum:"1m,3m,5m,15m,30m,1h,2h,4h,6h,12h,1D,1W,1M" doc:"Timeframe."`
	From       int64  `query:"from" minimum:"0" doc:"Oldest open time to include, Unix ms."`
	To         int64  `query:"to" minimum:"0" doc:"Newest open time to include, Unix ms. Defaults to the newest bar."`
	Limit      int    `query:"limit" minimum:"1" maximum:"5000" default:"2000" doc:"Maximum number of bars."`
}

type candlesOutput struct {
	Body candlePage
}

type candlePage struct {
	Bars            []bar  `json:"bars" doc:"Closed bars, oldest first: the newest ones in the range, up to limit."`
	NextTo          *int64 `json:"nextTo,omitempty" doc:"The to of the next older page, when the range has more bars."`
	HistoryComplete bool   `json:"historyComplete" doc:"Whether no bars exist before the first one returned."`
}

type bar struct {
	T   int64   `json:"t" doc:"Open time, Unix ms."`
	O   float64 `json:"o"`
	H   float64 `json:"h"`
	L   float64 `json:"l"`
	C   float64 `json:"c"`
	V   float64 `json:"v"`
	Rev int16   `json:"rev" doc:"Revision; a correction has a higher one."`
}

func registerCandles(api huma.API, repo marketdata.CandleRepository, instruments map[string]int32, log *slog.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "getCandles",
		Method:      http.MethodGet,
		Path:        "/api/v1/candles",
		Summary:     "Candle history",
		Description: "Returns stored closed bars, paged backwards in time with nextTo.",
		Tags:        []string{"Candles"},
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *candlesInput) (*candlesOutput, error) {
		id, ok := instruments[in.Instrument]
		if !ok {
			return nil, huma.Error404NotFound("unknown instrument " + in.Instrument)
		}
		tf, _ := marketdata.ParseTimeframe(in.TF) // validated by the enum
		end := int64(math.MaxInt64)
		if in.To > 0 && in.To < end {
			end = in.To + 1
		}
		// One bar more than asked for tells whether older bars remain.
		cs, err := repo.Candles(ctx, marketdata.CandleRange{
			InstrumentID: id, TF: tf, From: in.From, To: end, Limit: in.Limit + 1,
		})
		if err != nil {
			log.ErrorContext(ctx, "read candles", logx.KeyErr, err)
			return nil, huma.Error500InternalServerError("")
		}
		out := &candlesOutput{}
		if len(cs) > in.Limit {
			cs = cs[1:]
			next := cs[0].T - 1
			out.Body.NextTo = &next
		}
		out.Body.HistoryComplete = out.Body.NextTo == nil && in.From == 0
		out.Body.Bars = make([]bar, len(cs))
		for i, c := range cs {
			out.Body.Bars[i] = bar{c.T, c.O, c.H, c.L, c.C, c.V, c.Rev}
		}
		return out, nil
	})
}
