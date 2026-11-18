// SPDX-License-Identifier: AGPL-3.0-only

import { useEffect, useRef } from 'react';
import { createCandleChart } from '../../chart/engine/candle-chart';
import { subscribeCandles, wsURL } from '../../realtime/candles';

export function LiveChart({ instrument, tf }: { instrument: string; tf: string }) {
  const el = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!el.current) return;
    const chart = createCandleChart(el.current);
    const stop = subscribeCandles(wsURL(), instrument, tf, (c) => chart.update(c));
    return () => {
      stop();
      chart.destroy();
    };
  }, [instrument, tf]);
  return <div ref={el} className="h-[70vh] w-full" />;
}
