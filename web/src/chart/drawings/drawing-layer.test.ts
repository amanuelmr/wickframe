// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, test, vi } from 'vitest';
import { DrawingLayer } from './drawing-layer';
import { hitTest, type Projection, type TrendLine } from './trend-line';

// 1 px per second from T0 horizontally, 1 px per price unit vertically (price grows upward).
const T0 = Date.UTC(2026, 0, 5);
const proj: Projection = {
  timeToX: (t) => (t - T0) / 1000,
  priceToY: (p) => 1000 - p,
  xToTime: (x) => T0 + x * 1000,
  yToPrice: (y) => 1000 - y,
};
const at = (x: number, y: number) => ({ t: T0 + x * 1000, p: 1000 - y });

function layerWith(...lines: TrendLine[]) {
  let n = 0;
  const changed = vi.fn();
  const layer = new DrawingLayer(proj, changed, () => `id${++n}`);
  layer.lines.push(...lines);
  return { layer, changed };
}

const line = (id: string, ax: number, ay: number, bx: number, by: number): TrendLine => ({
  id,
  a: at(ax, ay),
  b: at(bx, by),
});

describe('hitTest', () => {
  const lines = [line('low', 0, 100, 200, 100), line('top', 100, 0, 100, 200)];

  test.each([
    ['an endpoint', 0, 103, { id: 'low', part: 'a' }],
    ['the other endpoint', 197, 100, { id: 'low', part: 'b' }],
    ['the segment', 50, 105, { id: 'low', part: 'line' }],
    ['the crossing, where the later line wins', 100, 100, { id: 'top', part: 'line' }],
    ['empty space', 50, 120, null],
    ['the extension beyond an endpoint', 220, 100, null],
  ])('finds %s', (_, x, y, want) => {
    expect(hitTest(lines, proj, x, y)).toEqual(want);
  });
});

describe('DrawingLayer', () => {
  test('creates a selected line with two clicks and consumes both', () => {
    const { layer } = layerWith();
    layer.startPlacing();
    expect(layer.pointerDown(10, 20)).toBe(true);
    expect(layer.lines).toHaveLength(0);
    expect(layer.pointerDown(110, 220)).toBe(true);
    expect(layer.lines).toEqual([line('id1', 10, 20, 110, 220)]);
    expect(layer.selectedId).toBe('id1');
    expect(layer.pending).toBeNull();
  });

  test('previews the second anchor while placing', () => {
    const { layer } = layerWith();
    layer.startPlacing();
    layer.pointerDown(10, 20);
    expect(layer.pointerMove(50, 60)).toBe(true);
    expect(layer.pending?.b).toEqual(at(50, 60));
  });

  test('drags one endpoint and leaves the other anchored', () => {
    const { layer } = layerWith(line('l', 0, 0, 100, 100));
    expect(layer.pointerDown(100, 100)).toBe(true);
    layer.pointerMove(150, 80);
    layer.pointerUp();
    expect(layer.lines[0]).toEqual(line('l', 0, 0, 150, 80));
  });

  test('drags the whole line by the pointer delta', () => {
    const { layer } = layerWith(line('l', 0, 0, 100, 100));
    layer.pointerDown(50, 50);
    layer.pointerMove(60, 30);
    layer.pointerMove(70, 40);
    layer.pointerUp();
    expect(layer.lines[0]).toEqual(line('l', 20, -10, 120, 90));
    expect(layer.pointerMove(500, 500)).toBe(false);
  });

  test('a click on empty space deselects and leaves the event to the chart', () => {
    const { layer, changed } = layerWith(line('l', 0, 0, 100, 100));
    layer.pointerDown(50, 50);
    layer.pointerUp();
    changed.mockClear();
    expect(layer.pointerDown(300, 10)).toBe(false);
    expect(layer.selectedId).toBeNull();
    expect(changed).toHaveBeenCalledOnce();
  });

  test.each(['Delete', 'Backspace'])('%s removes the selected line', (key) => {
    const { layer } = layerWith(line('a', 0, 0, 100, 0), line('b', 0, 50, 100, 50));
    layer.pointerDown(50, 50);
    expect(layer.keyDown(key)).toBe(true);
    expect(layer.lines.map((l) => l.id)).toEqual(['a']);
    expect(layer.keyDown(key)).toBe(false);
  });

  test('Escape cancels placing', () => {
    const { layer } = layerWith();
    layer.startPlacing();
    layer.pointerDown(10, 10);
    expect(layer.keyDown('Escape')).toBe(true);
    expect(layer.pending).toBeNull();
    expect(layer.pointerDown(20, 20)).toBe(false);
    expect(layer.lines).toHaveLength(0);
  });
});
