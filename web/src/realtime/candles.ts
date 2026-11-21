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

/**
 * Joins history with live updates: updates are held until the history has
 * loaded, then those at or after its last bar are applied, and older ones only
 * if they correct a bar on the chart.
 */
export class Stitcher {
  private held: Candle[] | null = [];
  private lastT = Number.NEGATIVE_INFINITY;
  private readonly revs = new Map<number, number>();

  /** apply gets historical true for a correction to a bar before the newest one. */
  constructor(private readonly apply: (c: Candle, historical: boolean) => void) {}

  live(c: Candle): void {
    if (this.held) {
      this.held.push(c);
      return;
    }
    const shown = this.revs.get(c.t);
    if (c.t > this.lastT || (c.t === this.lastT && c.rev >= (shown ?? 0))) {
      this.lastT = c.t;
      this.revs.set(c.t, c.rev);
      this.apply(c, false);
    } else if (shown !== undefined && c.rev > shown) {
      this.revs.set(c.t, c.rev);
      this.apply(c, true);
    }
  }

  /** Marks the history, oldest bar first, as loaded. */
  loaded(bars: { t: number; rev: number }[]): void {
    for (const b of bars) this.revs.set(b.t, b.rev);
    this.lastT = bars.at(-1)?.t ?? Number.NEGATIVE_INFINITY;
    const held = this.held ?? [];
    this.held = null;
    for (const c of held) this.live(c);
  }
}

export interface StreamOptions {
  url: string;
  instrument: string;
  tf: string;
  /** Loads and shows the closed bars up to lastClosedT, after every subscribe. */
  loadHistory(lastClosedT: number, signal: AbortSignal): Promise<{ t: number; rev: number }[]>;
  onCandle(c: Candle, historical: boolean): void;
}

const backoff = { base: 500, max: 30_000 };

/**
 * Streams one series until the returned function is called, reconnecting with
 * backoff when the connection drops.
 */
export function streamCandles(opts: StreamOptions): () => void {
  let stopped = false;
  let failures = 0;
  let ws: WebSocket;
  let retry: ReturnType<typeof setTimeout> | undefined;

  const connect = () => {
    const sock = new WebSocket(opts.url, PROTOCOL);
    ws = sock;
    const history = new AbortController();
    let stitch: Stitcher | undefined;
    sock.onopen = () =>
      sock.send(
        JSON.stringify({
          type: 'subscribe',
          sub: 'c1',
          channel: 'candles',
          instrument: opts.instrument,
          tf: opts.tf,
        }),
      );
    sock.onmessage = (e: MessageEvent<string>) => {
      let msg: { type?: string; lastClosedT?: number } | null;
      try {
        msg = JSON.parse(e.data);
      } catch {
        console.warn('wickframe: skipped a malformed WebSocket message', e.data);
        return;
      }
      if (msg?.type === 'subscribed') {
        failures = 0;
        const s = new Stitcher(opts.onCandle);
        stitch = s;
        opts.loadHistory(msg.lastClosedT ?? 0, history.signal).then(
          (bars) => s.loaded(bars),
          () => sock.close(4000, 'history failed'),
        );
      } else if (msg?.type === 'candle') {
        stitch?.live(msg as Candle);
      }
    };
    sock.onclose = (e: CloseEvent) => {
      history.abort();
      // 1008 needs the user to sign in again, which a retry cannot do.
      if (stopped || e.code === 1008) return;
      const delay = Math.random() * Math.min(backoff.max, backoff.base * 2 ** failures);
      failures++;
      retry = setTimeout(connect, delay);
    };
  };
  connect();

  return () => {
    stopped = true;
    clearTimeout(retry);
    ws.close(1000);
  };
}

export function wsURL(loc: Location = window.location): string {
  return `${loc.protocol === 'https:' ? 'wss' : 'ws'}://${loc.host}/api/v1/ws`;
}
