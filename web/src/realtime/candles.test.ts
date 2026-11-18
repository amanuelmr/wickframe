// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, expect, test, vi } from 'vitest';
import { PROTOCOL, subscribeCandles, wsURL } from './candles';

class FakeSocket {
  static last: FakeSocket;
  sent: string[] = [];
  closedWith?: number;
  onopen?: () => void;
  onmessage?: (e: { data: string }) => void;
  constructor(
    readonly url: string,
    readonly protocol: string,
  ) {
    FakeSocket.last = this;
  }
  send(data: string) {
    this.sent.push(data);
  }
  close(code: number) {
    this.closedWith = code;
  }
}

afterEach(() => vi.unstubAllGlobals());

test('subscribes on open and passes on only candle messages', () => {
  vi.stubGlobal('WebSocket', FakeSocket);
  const got: unknown[] = [];
  const stop = subscribeCandles('ws://x/api/v1/ws', 'fake:DEMO', '1m', (c) => got.push(c));
  const ws = FakeSocket.last;
  expect(ws.protocol).toBe(PROTOCOL);

  ws.onopen?.();
  expect(JSON.parse(ws.sent[0] ?? '')).toEqual({
    type: 'subscribe',
    sub: 'c1',
    channel: 'candles',
    instrument: 'fake:DEMO',
    tf: '1m',
  });

  const candle = {
    type: 'candle',
    sub: 'c1',
    seq: 1,
    t: 60000,
    o: 1,
    h: 2,
    l: 1,
    c: 2,
    v: 0,
    state: 'forming',
    rev: 0,
  };
  ws.onmessage?.({
    data: JSON.stringify({ type: 'subscribed', sub: 'c1', seq: 0, lastClosedT: 0 }),
  });
  ws.onmessage?.({ data: JSON.stringify(candle) });
  expect(got).toEqual([candle]);

  stop();
  expect(ws.closedWith).toBe(1000);
});

test('uses wss on https pages', () => {
  expect(wsURL({ protocol: 'https:', host: 'charts.example' } as Location)).toBe(
    'wss://charts.example/api/v1/ws',
  );
  expect(wsURL({ protocol: 'http:', host: '127.0.0.1:7317' } as Location)).toBe(
    'ws://127.0.0.1:7317/api/v1/ws',
  );
});
