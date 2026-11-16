// SPDX-License-Identifier: AGPL-3.0-only

import {
  CandlestickSeries,
  createChart,
  type IChartApi,
  type IPrimitivePaneView,
  type ISeriesApi,
  type ISeriesPrimitive,
  type Logical,
  PriceScaleMode,
  type PrimitiveHoveredItem,
  type SeriesAttachedParameter,
  type Time,
  type UTCTimestamp,
} from 'lightweight-charts';
import { type Bar, logicalToTime, timeToLogical } from '../model/bars';
import type { ChartEngine, Overlay, ScaleMode } from './chart-engine';

// Lightweight Charts uses UTC seconds; the rest of Wickframe uses UTC ms.
const toTime = (ms: number) => (ms / 1000) as UTCTimestamp;

class OverlayPrimitive implements ISeriesPrimitive<Time> {
  private params: SeriesAttachedParameter<Time> | null = null;
  private readonly views: readonly IPrimitivePaneView[];

  constructor(private readonly overlay: Overlay) {
    this.views = [
      {
        zOrder: () => 'top',
        renderer: () => ({
          draw: (target) => target.useMediaCoordinateSpace(({ context }) => overlay.draw(context)),
        }),
      },
    ];
  }

  attached(params: SeriesAttachedParameter<Time>): void {
    this.params = params;
  }

  detached(): void {
    this.params = null;
  }

  paneViews(): readonly IPrimitivePaneView[] {
    return this.views;
  }

  requestUpdate(): void {
    this.params?.requestUpdate();
  }

  hitTest(x: number, y: number): PrimitiveHoveredItem | null {
    const cursorStyle = this.overlay.cursorAt?.(x, y);
    return cursorStyle ? { externalId: 'overlay', zOrder: 'top', cursorStyle } : null;
  }
}

export class LwcEngine implements ChartEngine {
  readonly chart: IChartApi;
  private readonly candles: ISeriesApi<'Candlestick'>;
  private readonly overlays = new Set<OverlayPrimitive>();
  private bars: readonly Bar[] = [];
  private intervalMs = 60_000;

  constructor(container: HTMLElement) {
    // attributionLogo keeps its default (true); the licence requires it (ADR-0005 rule 3).
    this.chart = createChart(container, { autoSize: true });
    this.candles = this.chart.addSeries(CandlestickSeries);
  }

  setBars(bars: readonly Bar[], intervalMs: number): void {
    this.bars = bars;
    this.intervalMs = intervalMs;
    this.candles.setData(
      bars.map((b) => ({ time: toTime(b.t), open: b.o, high: b.h, low: b.l, close: b.c })),
    );
  }

  setScaleMode(mode: ScaleMode): void {
    this.chart.priceScale('right').applyOptions({
      mode: mode === 'log' ? PriceScaleMode.Logarithmic : PriceScaleMode.Normal,
    });
  }

  addOverlay(overlay: Overlay): () => void {
    const primitive = new OverlayPrimitive(overlay);
    this.candles.attachPrimitive(primitive);
    this.overlays.add(primitive);
    return () => {
      this.candles.detachPrimitive(primitive);
      this.overlays.delete(primitive);
    };
  }

  requestRedraw(): void {
    for (const o of this.overlays) o.requestUpdate();
  }

  timeToX(t: number): number | null {
    // timeScale().timeToCoordinate returns null between bars and beyond the data.
    const logical = timeToLogical(this.bars, t, this.intervalMs);
    if (logical === null) return null;
    // logicalToCoordinate returns 0 for fractional indexes (5.3.0), but the mapping is linear,
    // so interpolate between the two whole indexes around it.
    const ts = this.chart.timeScale();
    const i = Math.floor(logical);
    const x0 = ts.logicalToCoordinate(i as Logical);
    const x1 = ts.logicalToCoordinate((i + 1) as Logical);
    return x0 === null || x1 === null ? null : x0 + (logical - i) * (x1 - x0);
  }

  priceToY(p: number): number | null {
    return this.candles.priceToCoordinate(p);
  }

  xToTime(x: number): number | null {
    // coordinateToLogical rounds up to a whole index, which would snap every anchor to a bar.
    const ts = this.chart.timeScale();
    const i = ts.coordinateToLogical(x);
    if (i === null) return null;
    const xi = ts.logicalToCoordinate(i);
    const next = ts.logicalToCoordinate((i + 1) as Logical);
    if (xi === null || next === null) return null;
    return logicalToTime(this.bars, i + (x - xi) / (next - xi), this.intervalMs);
  }

  yToPrice(y: number): number | null {
    return this.candles.coordinateToPrice(y);
  }

  setPanZoomEnabled(enabled: boolean): void {
    this.chart.applyOptions({ handleScroll: enabled, handleScale: enabled });
  }

  visibleRange(): { from: number; to: number } | null {
    return this.chart.timeScale().getVisibleLogicalRange();
  }

  setVisibleRange(range: { from: number; to: number }): void {
    this.chart.timeScale().setVisibleLogicalRange(range);
  }

  destroy(): void {
    this.chart.remove();
  }
}
