# Glossary

| Term | Definition |
|---|---|
| **ADR** | Architecture Decision Record. A short document capturing one significant decision, its context and its consequences ([ADR-0001](../adr/0001-record-architecture-decisions.md)). |
| **Aggregation** | Building candles of a coarser timeframe from a finer one (for example 1m → 5m). |
| **Alert** | A user-defined condition on an instrument and timeframe that, when met, records an event and sends notifications. |
| **Anchor** | A (time, price) point that defines a drawing's position. |
| **arc42** | A template for documenting software architecture, used to structure these docs. |
| **Backfill** | Fetching historical candles to fill a gap or extend history. |
| **BYOD** | Bring your own data: users connect their own market-data sources ([ADR-0006](../adr/0006-bring-your-own-market-data.md)). |
| **C4 model** | A hierarchical way to diagram software: Context, Containers, Components, Code. |
| **Candle (bar)** | An OHLCV summary of trading in one time bucket, identified by its open time. |
| **Closed / forming** | A *forming* candle is the current, still-changing bar. A *closed* candle is final (apart from source corrections). |
| **Conflation** | Replacing a queued, not-yet-sent update with a newer one for the same key, so slow consumers get the latest state without unbounded queues. |
| **Connector** | An adapter translating a specific data source's API into Wickframe's market-data model. |
| **Coverage index** | Records which time ranges of a candle series are known to be complete locally. |
| **DCO** | Developer Certificate of Origin, a per-commit sign-off certifying the contributor's right to submit the code. |
| **Derived timeframe** | A timeframe Wickframe aggregates itself, rather than fetching it natively. |
| **expr** | `expr-lang/expr`, the Go expression language used for alert conditions. |
| **Gap** | A period in a candle series where data is missing because of an outage or source limitation (not merely zero trades). |
| **Hexagonal architecture** | Ports and adapters: domain logic defines interfaces (ports); I/O implementations (adapters) plug into them. |
| **Instrument key** | The stable public identifier `source:symbol`, for example `binance:BTCUSDT`. |
| **Lossless subscription** | A bus subscription that never drops events (back-pressure instead). |
| **MADR** | Markdown Architectural Decision Records, the ADR template used here. |
| **OHLCV** | Open, High, Low, Close, Volume. |
| **Outbox (transactional)** | A table written in the same transaction as a business event, from which a dispatcher later delivers messages reliably. |
| **Personal mode / server mode** | The two operating modes of one binary ([overview §5](00-overview.md#5-operating-modes)). |
| **Problem Details** | RFC 9457 standard JSON error format. |
| **Role (`--target`)** | A group of backend modules that can be started selectively ([ADR-0003](../adr/0003-go-modular-monolith.md)). |
| **Series key** | The identity of a computed indicator series: instrument, timeframe, indicator, version and parameters. |
| **Session** | (Market) the trading hours of an exchange. (Auth) a server-side login session. Context makes clear which. |
| **SSRF** | Server-Side Request Forgery: tricking a server into making requests to internal resources. |
| **Tick size** | The minimum price increment of an instrument. |
| **Timeframe** | The duration of one candle (1m, 1h, 1D, …). |
| **UUIDv7** | A time-ordered UUID (RFC 9562), used for most identifiers. |
| **Warm-up** | The number of bars an indicator needs before its output is valid. |
