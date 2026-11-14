# Wickframe

**Free, open-source, self-hostable charting and technical analysis.**
Professional charts, indicators, drawings and 24/7 alerts that run on your own
machine or server, using market data you connect yourself.

> **Status: pre-alpha (architecture phase).** The architecture and plans are
> documented below. Code is being built toward the first release, v0.1. Nothing is
> ready for use yet. Star or watch the repository to follow along.

## Why Wickframe?

- **Own your tools:** no subscriptions, and no paywalled indicators or alerts.
- **Own your data:** layouts, drawings, alerts and watchlists live in your own
  database, exportable in open formats. No telemetry.
- **Bring your own data:** connect exchanges and brokers directly with your own
  access. Wickframe never resells or redistributes market data.
- **Correct by design:** every built-in indicator is verified against a reference
  implementation, and charts and alerts share one indicator engine.
- **Easy to run:** one binary or one container. SQLite for personal use, PostgreSQL
  for servers.

## Planned features

| | |
|---|---|
| Charts | Candles, bars, line and area charts; pan and zoom; multi-pane indicators; watchlists; light and dark themes |
| Drawings | Trend lines, rays, horizontal and vertical lines, rectangles, Fibonacci tools, channels, measures, text |
| Indicators | 20+ verified built-ins (MA family, RSI, MACD, Bollinger, ATR, ADX, Ichimoku, VWAP, …); custom scripts later |
| Alerts | Price and indicator conditions, 24/7 in server mode, via webhook (signed), email, Telegram, Discord, ntfy |
| Data | Binance, Kraken and Coinbase at launch; Alpaca (US stocks), CSV import, and more to follow |
| Platform | REST API (OpenAPI 3.1), WebSocket streaming, multi-user server mode, desktop app |

See the [roadmap](docs/product/roadmap.md) for milestones.

## Documentation

| | |
|---|---|
| [Product vision](docs/product/vision.md) | Problem, users, goals and non-goals |
| [Requirements](docs/product/requirements.md) | Functional and non-functional requirements |
| [Architecture overview](docs/architecture/00-overview.md) | Start here for the technical design |
| [Architecture decisions](docs/adr/README.md) | Why the system is built this way |
| [Engineering guides](docs/README.md#engineering) | Workflow, layout, standards, releases |

## Technology

Go 1.27 (single binary) · SQLite and PostgreSQL · React 19 + TypeScript 7 + Vite 8 ·
TradingView Lightweight Charts™ v5 (chart rendering) · WebSocket · OpenAPI 3.1 ·
Wails v3 (desktop).

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) and our
[Code of Conduct](CODE_OF_CONDUCT.md). Security issues: see [SECURITY.md](SECURITY.md).

## License

Wickframe is licensed under the [GNU Affero General Public License v3.0 only](LICENSE)
([why](docs/adr/0002-license-agpl-3.0.md)).

Chart rendering uses TradingView Lightweight Charts™, Copyright (с) 2025
TradingView, Inc., <https://www.tradingview.com/>, under the Apache License 2.0.
Wickframe is an independent project and is not affiliated with or endorsed by
TradingView.

## Disclaimer

Wickframe is a charting and analysis tool. It does not provide financial advice,
and does not place trades. Market data accuracy depends on your data sources.
Use at your own risk.
