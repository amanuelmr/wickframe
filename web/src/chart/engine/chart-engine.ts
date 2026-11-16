// SPDX-License-Identifier: AGPL-3.0-only

import type { Bar } from '../model/bars';

export type ScaleMode = 'linear' | 'log';

export interface Overlay {
  /** Draws above the candles on every chart repaint, in CSS pixels. */
  draw(ctx: CanvasRenderingContext2D): void;
  /** CSS cursor for the pointer at (x, y), or null to keep the chart's own. */
  cursorAt?(x: number, y: number): string | null;
}

/**
 * The only way the app talks to a chart library (ADR-0005). Times are UTC ms, prices are
 * prices, and coordinates are CSS pixels of the price pane; no library types cross it.
 */
export interface ChartEngine {
  setBars(bars: readonly Bar[], intervalMs: number): void;
  setScaleMode(mode: ScaleMode): void;
  /** Returns a function that removes the overlay. */
  addOverlay(overlay: Overlay): () => void;
  requestRedraw(): void;
  /** Works between bars, before the first bar and in the future. */
  timeToX(t: number): number | null;
  priceToY(p: number): number | null;
  xToTime(x: number): number | null;
  yToPrice(y: number): number | null;
  /** Off while a drawing owns the pointer, so a drag does not also pan the chart. */
  setPanZoomEnabled(enabled: boolean): void;
  /** Visible range in fractional bar indices. */
  visibleRange(): { from: number; to: number } | null;
  setVisibleRange(range: { from: number; to: number }): void;
  destroy(): void;
}
