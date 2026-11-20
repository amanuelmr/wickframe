// SPDX-License-Identifier: AGPL-3.0-only

import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { type Candle, PROTOCOL, Stitcher, streamCandles, wsURL } from './candles';

const bar = (t: number, state: Candle['state'] = 'closed', rev = 0): Candle => ({
  t,
  o: 1,
  h: 2,
  l: 1,
  c: 2,
  v: 0,
  state,
  rev,
});

describe('Stitcher', () => {
  test('holds updates that arrive before the history, then applies those at or after its last bar', () => {
    const got: Candle[] = [];
    const s = new Stitcher((c) => got.push(c));
    s.live(bar(60_000, 'forming'));
    s.live(bar(0, 'closed', 1)); // a correction older than the last history bar
    s.live(bar(120_000, 'forming'));
    expect(got).toEqual([]);

    s.loaded(60_000);
    expect(got).toEqual([bar(60_000, 'forming'), bar(120_000, 'forming')]);
  });

  test('drops a later update that is older than the last history bar', () => {
    const got: Candle[] = [];
    const s = new Stitcher((c) => got.push(c));
    s.loaded(120_000);
    s.live(bar(60_000, 'closed', 1));
    s.live(bar(120_000, 'closed', 1));
    s.live(bar(180_000, 'forming'));
    expect(got).toEqual([bar(120_000, 'closed', 1), bar(180_000, 'forming')]);
  });

  test('applies every update after an empty history', () => {
    const got: Candle[] = [];
    const s = new Stitcher((c) => got.push(c));
    s.live(bar(0, 'forming'));
    s.loaded(undefined);
    s.live(bar(60_000, 'forming'));
    expect(got).toEqual([bar(0, 'forming'), bar(60_000, 'forming')]);
  });
});

class FakeSocket {
  static all: FakeSocket[] = [];
  sent: unknown[] = [];
  closedWith?: number;
  onopen?: () => void;
  onmessage?: (e: { data: string }) => void;
  onclose?: (e: { code: number }) => void;
  constructor(
    readonly url: string,
    readonly protocol: string,
  ) {
    FakeSocket.all.push(this);
  }
  send(data: string) {
    this.sent.push(JSON.parse(data));
  }
  close(code: number) {
    this.closedWith = code;
    this.onclose?.({ code });
  }
  receive(msg: object) {
    this.onmessage?.({ data: JSON.stringify(msg) });
  }
}

describe('streamCandles', () => {
  let history: { to: number; resolve: (bars: { t: number }[]) => void }[];
  let got: Candle[];
  let stop: () => void;

  beforeEach(() => {
    vi.stubGlobal('WebSocket', FakeSocket);
    vi.useFakeTimers();
    vi.spyOn(Math, 'random').mockReturnValue(1);
    FakeSocket.all = [];
    history = [];
    got = [];
    stop = streamCandles({
      url: 'ws://x/api/v1/ws',
      instrument: 'fake:DEMO',
      tf: '1m',
      loadHistory: (to) => new Promise((resolve) => history.push({ to, resolve })),
      onCandle: (c) => got.push(c),
    });
  });

  afterEach(() => {
    stop();
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  const subscribe = (ws: FakeSocket, lastClosedT: number) => {
    ws.onopen?.();
    ws.receive({ type: 'subscribed', sub: 'c1', seq: 0, lastClosedT });
  };
  const candle = (c: Candle) => ({ type: 'candle', sub: 'c1', seq: 1, ...c });

  test('subscribes, loads history up to lastClosedT, then applies the held updates', async () => {
    const ws = FakeSocket.all[0] as FakeSocket;
    expect(ws.protocol).toBe(PROTOCOL);
    subscribe(ws, 60_000);
    expect(ws.sent).toEqual([
      { type: 'subscribe', sub: 'c1', channel: 'candles', instrument: 'fake:DEMO', tf: '1m' },
    ]);
    expect(history.map((h) => h.to)).toEqual([60_000]);

    ws.receive(candle(bar(120_000, 'forming')));
    expect(got).toEqual([]);
    history[0]?.resolve([{ t: 0 }, { t: 60_000 }]);
    await vi.runAllTimersAsync();
    expect(got).toEqual([expect.objectContaining(bar(120_000, 'forming'))]);
  });

  test.each([1013, 1011, 1006])(
    'reconnects after close code %i and reloads history',
    async (code) => {
      subscribe(FakeSocket.all[0] as FakeSocket, 60_000);
      (FakeSocket.all[0] as FakeSocket).onclose?.({ code });

      await vi.advanceTimersByTimeAsync(499);
      expect(FakeSocket.all).toHaveLength(1);
      await vi.advanceTimersByTimeAsync(1);
      expect(FakeSocket.all).toHaveLength(2);

      subscribe(FakeSocket.all[1] as FakeSocket, 120_000);
      expect(history.map((h) => h.to)).toEqual([60_000, 120_000]);
    },
  );

  test('backs off longer after each failed attempt', async () => {
    for (const wait of [500, 1000, 2000]) {
      (FakeSocket.all.at(-1) as FakeSocket).onclose?.({ code: 1006 });
      await vi.advanceTimersByTimeAsync(wait - 1);
      const before = FakeSocket.all.length;
      await vi.advanceTimersByTimeAsync(1);
      expect(FakeSocket.all.length).toBe(before + 1);
    }
  });

  test('does not reconnect after 1008 or after stop', async () => {
    (FakeSocket.all[0] as FakeSocket).onclose?.({ code: 1008 });
    await vi.advanceTimersByTimeAsync(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });

  test('reconnects when the history fails to load', async () => {
    const failing = streamCandles({
      url: 'ws://x/api/v1/ws',
      instrument: 'fake:DEMO',
      tf: '1m',
      loadHistory: () => Promise.reject(new Error('HTTP 500')),
      onCandle: () => {},
    });
    const ws = FakeSocket.all.at(-1) as FakeSocket;
    subscribe(ws, 0);
    await vi.advanceTimersByTimeAsync(0);
    expect(ws.closedWith).toBe(4000);
    await vi.advanceTimersByTimeAsync(500);
    expect(FakeSocket.all.at(-1)).not.toBe(ws);
    failing();
  });

  test('stop closes normally without reconnecting', async () => {
    stop();
    expect((FakeSocket.all[0] as FakeSocket).closedWith).toBe(1000);
    await vi.advanceTimersByTimeAsync(60_000);
    expect(FakeSocket.all).toHaveLength(1);
  });
});

test('uses wss on https pages', () => {
  expect(wsURL({ protocol: 'https:', host: 'charts.example' } as Location)).toBe(
    'wss://charts.example/api/v1/ws',
  );
  expect(wsURL({ protocol: 'http:', host: '127.0.0.1:7317' } as Location)).toBe(
    'ws://127.0.0.1:7317/api/v1/ws',
  );
});
