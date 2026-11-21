// SPDX-License-Identifier: AGPL-3.0-only

import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { type GetCandlesTf, getGetCandlesQueryOptions } from '../../api/generated/wickframe';
import { createCandleChart } from '../../chart/engine/candle-chart';
import { streamCandles, wsURL } from '../../realtime/candles';

export function LiveChart({ instrument, tf }: { instrument: string; tf: GetCandlesTf }) {
  const el = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();
  useEffect(() => {
    if (!el.current) return;
    const chart = createCandleChart(el.current);
    const stop = streamCandles({
      url: wsURL(),
      instrument,
      tf,
      loadHistory: async (to, signal) => {
        const params = { instrument, tf, to, limit: 2000 };
        const page = await queryClient.fetchQuery(
          getGetCandlesQueryOptions(params, { request: { signal } }),
        );
        chart.setData(page.bars);
        return page.bars;
      },
      onCandle: (c, historical) => chart.update(c, historical),
    });
    return () => {
      stop();
      chart.destroy();
    };
  }, [instrument, tf, queryClient]);
  return <div ref={el} className="h-[70vh] w-full" />;
}
