# Binance fixtures

Synthetic. Every price, volume, trade count and IP address here is made up;
nothing was recorded from Binance (14-quality §4, ADR-0006 rule 6).

The files follow the wire format in Binance's spot API documentation, read on
2026-11-21:

| File | Format |
|---|---|
| `klines.json` | `GET /api/v3/klines` response: arrays of open time, open, high, low, close, volume, close time, quote volume, trades, taker buy base volume, taker buy quote volume, ignore |
| `error_429.json` | Body of a `429` (request weight exceeded), sent with `Retry-After` |
| `error_418.json` | Body of a `418` (IP banned), sent with `Retry-After` |
| `kline_stream.jsonl` | Combined-stream messages (`/stream?streams=…`) for `btcusdt@kline_1m`, one per line: two forming updates, the close, then the next bar forming |

The tests generate longer kline series in the same format to exercise paging.
