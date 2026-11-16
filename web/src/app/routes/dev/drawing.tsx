// SPDX-License-Identifier: AGPL-3.0-only

import { createFileRoute, notFound } from '@tanstack/react-router';
import { useEffect, useRef, useState } from 'react';
import { DrawingLayer } from '../../../chart/drawings/drawing-layer';
import { drawTrendLines } from '../../../chart/drawings/render';
import type { Part } from '../../../chart/drawings/trend-line';
import { LwcEngine } from '../../../chart/engine/lwc-engine';
import type { Bar } from '../../../chart/model/bars';

export const Route = createFileRoute('/dev/drawing')({
  beforeLoad: () => {
    if (!import.meta.env.DEV) throw notFound();
  },
  component: DrawingSpike,
});

const HOUR = 3_600_000;
const CURSOR: Record<Part, string> = { a: 'grab', b: 'grab', line: 'move' };

/** Hourly bars on weekdays only, so every weekend is a gap in time. Deterministic. */
function fakeBars(count: number): Bar[] {
  let seed = 42;
  const rand = () => {
    seed = (seed * 1_664_525 + 1_013_904_223) % 2 ** 32;
    return seed / 2 ** 32 - 0.5;
  };
  const bars: Bar[] = [];
  let t = Date.UTC(2025, 0, 6); // a Monday
  let price = 100;
  while (bars.length < count) {
    const day = new Date(t).getUTCDay();
    if (day !== 0 && day !== 6) {
      const o = price;
      const c = o * Math.exp(rand() * 0.02 + 0.0002);
      bars.push({
        t,
        o,
        c,
        h: Math.max(o, c) * (1 + Math.abs(rand()) * 0.01),
        l: Math.min(o, c) * (1 - Math.abs(rand()) * 0.01),
      });
      price = c;
    }
    t += HOUR;
  }
  return bars;
}

function percentile(values: number[], q: number): number {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.min(sorted.length - 1, Math.floor(q * sorted.length))] ?? 0;
}

function DrawingSpike() {
  const host = useRef<HTMLDivElement>(null);
  const [layer, setLayer] = useState<DrawingLayer | null>(null);
  const [log, setLog] = useState(false);
  const engineRef = useRef<LwcEngine | null>(null);

  useEffect(() => {
    const el = host.current;
    if (!el) return;
    const engine = new LwcEngine(el);
    const bars = fakeBars(10_000);
    engine.setBars(bars, HOUR);
    const layer = new DrawingLayer(engine, () => engine.requestRedraw());
    const drawTimes: number[] = [];
    const removeOverlay = engine.addOverlay({
      draw: (ctx) => {
        const start = performance.now();
        drawTrendLines(ctx, engine, layer);
        drawTimes.push(performance.now() - start);
      },
      cursorAt: (x, y) => {
        const hit = layer.hover(x, y);
        return hit ? CURSOR[hit.part] : null;
      },
    });

    const local = (e: PointerEvent) => {
      const r = el.getBoundingClientRect();
      return [e.clientX - r.left, e.clientY - r.top] as const;
    };
    const down = (e: PointerEvent) => {
      el.focus();
      if (!layer.pointerDown(...local(e))) return;
      // preventDefault stops the compatibility mouse events the chart pans with; touch events
      // are separate, so pan and zoom are switched off until the pointer is released.
      e.preventDefault();
      engine.setPanZoomEnabled(false);
      el.setPointerCapture(e.pointerId);
    };
    const move = (e: PointerEvent) => {
      if (layer.pointerMove(...local(e))) e.preventDefault();
    };
    const up = (e: PointerEvent) => {
      layer.pointerUp();
      engine.setPanZoomEnabled(true);
      if (el.hasPointerCapture(e.pointerId)) el.releasePointerCapture(e.pointerId);
    };
    const key = (e: KeyboardEvent) => {
      if (layer.keyDown(e.key)) e.preventDefault();
    };
    el.addEventListener('pointerdown', down, { capture: true });
    el.addEventListener('pointermove', move, { capture: true });
    el.addEventListener('pointerup', up, { capture: true });
    el.addEventListener('pointercancel', up, { capture: true });
    el.addEventListener('keydown', key);

    const ts = engine.chart.timeScale();
    Object.assign(window, {
      __spike: {
        engine,
        layer,
        bars,
        /** Max |x| difference between our projection and the library's, over loaded bars on screen. */
        anchorError() {
          const r = engine.visibleRange();
          if (!r) return null;
          let max = 0;
          for (let i = Math.max(0, Math.ceil(r.from)); i <= Math.min(bars.length - 1, r.to); i++) {
            const t = (bars[i] as Bar).t;
            const native = ts.timeToCoordinate((t / 1000) as never);
            const ours = engine.timeToX(t);
            if (native !== null && ours !== null) max = Math.max(max, Math.abs(native - ours));
          }
          return max;
        },
        nativeTimeToX: (t: number) => ts.timeToCoordinate((t / 1000) as never),
        addLines(n: number) {
          const r = engine.visibleRange();
          if (!r) return;
          for (let k = 0; k < n; k++) {
            const i = Math.floor(r.from + ((r.to - r.from) * k) / n);
            const a = bars[Math.max(0, Math.min(bars.length - 1, i))] as Bar;
            const b = bars[Math.max(0, Math.min(bars.length - 1, i + 20))] as Bar;
            layer.lines.push({ id: `bulk${k}`, a: { t: a.t, p: a.l }, b: { t: b.t, p: b.h } });
          }
          engine.requestRedraw();
        },
        /** Pans one bar per frame for `frames` frames and reports frame intervals and draw cost. */
        measure(frames = 300): Promise<Record<string, number>> {
          drawTimes.length = 0;
          const intervals: number[] = [];
          return new Promise((resolve) => {
            let last = performance.now();
            let n = 0;
            const step = (now: number) => {
              intervals.push(now - last);
              last = now;
              const r = engine.visibleRange();
              if (r)
                engine.setVisibleRange({
                  from: r.from + (n < frames / 2 ? 1 : -1),
                  to: r.to + (n < frames / 2 ? 1 : -1),
                });
              if (++n < frames) requestAnimationFrame(step);
              else
                resolve({
                  lines: layer.lines.length,
                  frameP50: percentile(intervals.slice(1), 0.5),
                  frameP95: percentile(intervals.slice(1), 0.95),
                  frameMax: Math.max(...intervals.slice(1)),
                  drawP50: percentile(drawTimes, 0.5),
                  drawP95: percentile(drawTimes, 0.95),
                  draws: drawTimes.length,
                });
            };
            requestAnimationFrame(step);
          });
        },
      },
    });

    engineRef.current = engine;
    setLayer(layer);
    return () => {
      el.removeEventListener('pointerdown', down, { capture: true });
      el.removeEventListener('pointermove', move, { capture: true });
      el.removeEventListener('pointerup', up, { capture: true });
      el.removeEventListener('pointercancel', up, { capture: true });
      el.removeEventListener('keydown', key);
      removeOverlay();
      engine.destroy();
      engineRef.current = null;
    };
  }, []);

  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-2">
        <button type="button" className="border px-2" onClick={() => layer?.startPlacing()}>
          Trend line
        </button>
        <button
          type="button"
          className="border px-2"
          onClick={() => {
            engineRef.current?.setScaleMode(log ? 'linear' : 'log');
            setLog(!log);
          }}
        >
          {log ? 'Linear scale' : 'Log scale'}
        </button>
      </div>
      {/* biome-ignore lint/a11y/noNoninteractiveTabindex: the chart takes keyboard focus for Delete and Escape. */}
      <div ref={host} tabIndex={0} className="h-[70vh] w-full" style={{ touchAction: 'none' }} />
    </div>
  );
}
