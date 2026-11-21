// SPDX-License-Identifier: AGPL-3.0-only

import { CandlestickSeries, createChart, type UTCTimestamp } from 'lightweight-charts';

export interface Bar {
  t: number; // open time, Unix ms UTC
  o: number;
  h: number;
  l: number;
  c: number;
}

export interface CandleChart {
  /** Replaces every bar. */
  setData(bars: Bar[]): void;
  /** Updates the last bar or appends a newer one; historical updates an older bar. */
  update(bar: Bar, historical?: boolean): void;
  destroy(): void;
}

const toLW = (b: Bar) => ({
  time: (b.t / 1000) as UTCTimestamp,
  open: b.o,
  high: b.h,
  low: b.l,
  close: b.c,
});

export function createCandleChart(el: HTMLElement): CandleChart {
  const chart = createChart(el, { autoSize: true, timeScale: { timeVisible: true } });
  const series = chart.addSeries(CandlestickSeries);
  return {
    setData: (bars) => series.setData(bars.map(toLW)),
    update: (b, historical) => series.update(toLW(b), historical),
    destroy: () => chart.remove(),
  };
}
