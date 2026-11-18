// SPDX-License-Identifier: AGPL-3.0-only

export const PROTOCOL = 'wickframe.v1';

export interface Candle {
  t: number;
  o: number;
  h: number;
  l: number;
  c: number;
  v: number;
  state: 'forming' | 'closed';
  rev: number;
}

/** Streams candle updates for one instrument and timeframe until the returned function is called. */
export function subscribeCandles(
  url: string,
  instrument: string,
  tf: string,
  onCandle: (c: Candle) => void,
): () => void {
  const ws = new WebSocket(url, PROTOCOL);
  ws.onopen = () =>
    ws.send(JSON.stringify({ type: 'subscribe', sub: 'c1', channel: 'candles', instrument, tf }));
  ws.onmessage = (e: MessageEvent<string>) => {
    const msg = JSON.parse(e.data);
    if (msg.type === 'candle') onCandle(msg);
  };
  return () => ws.close(1000);
}

export function wsURL(loc: Location = window.location): string {
  return `${loc.protocol === 'https:' ? 'wss' : 'ws'}://${loc.host}/api/v1/ws`;
}
