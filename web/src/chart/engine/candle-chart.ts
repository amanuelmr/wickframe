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
  update(bar: Bar): void;
  destroy(): void;
}

export function createCandleChart(el: HTMLElement): CandleChart {
  const chart = createChart(el, { autoSize: true, timeScale: { timeVisible: true } });
  const series = chart.addSeries(CandlestickSeries);
  return {
    update: (b) =>
      series.update({
        time: (b.t / 1000) as UTCTimestamp,
        open: b.o,
        high: b.h,
        low: b.l,
        close: b.c,
      }),
    destroy: () => chart.remove(),
  };
}
