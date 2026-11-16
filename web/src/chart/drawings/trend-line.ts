// SPDX-License-Identifier: AGPL-3.0-only

/** A drawing anchor in domain space: UTC ms and price (FR-CHART-07). */
export interface Anchor {
  t: number;
  p: number;
}

export interface TrendLine {
  id: string;
  a: Anchor;
  b: Anchor;
}

/** Converts between domain space and CSS pixels of the price pane. */
export interface Projection {
  timeToX(t: number): number | null;
  priceToY(p: number): number | null;
  xToTime(x: number): number | null;
  yToPrice(y: number): number | null;
}

export type Part = 'a' | 'b' | 'line';

export interface Hit {
  id: string;
  part: Part;
}

export const HIT_TOLERANCE_PX = 6;

export function distanceToSegment(
  px: number,
  py: number,
  x1: number,
  y1: number,
  x2: number,
  y2: number,
): number {
  const dx = x2 - x1;
  const dy = y2 - y1;
  const len2 = dx * dx + dy * dy;
  const k = len2 === 0 ? 0 : Math.max(0, Math.min(1, ((px - x1) * dx + (py - y1) * dy) / len2));
  return Math.hypot(px - (x1 + k * dx), py - (y1 + k * dy));
}

/** Pixel position of an anchor, or null when the engine cannot place it (no data). */
export function project(proj: Projection, a: Anchor): { x: number; y: number } | null {
  const x = proj.timeToX(a.t);
  const y = proj.priceToY(a.p);
  return x === null || y === null ? null : { x, y };
}

/**
 * The topmost line under (x, y). Endpoints win over the segment so a short line can still be
 * resized; later lines are drawn on top, so they are tested first.
 */
export function hitTest(
  lines: readonly TrendLine[],
  proj: Projection,
  x: number,
  y: number,
  tolerance = HIT_TOLERANCE_PX,
): Hit | null {
  for (let i = lines.length - 1; i >= 0; i--) {
    const line = lines[i] as TrendLine;
    const a = project(proj, line.a);
    const b = project(proj, line.b);
    if (!a || !b) continue;
    if (Math.hypot(x - a.x, y - a.y) <= tolerance) return { id: line.id, part: 'a' };
    if (Math.hypot(x - b.x, y - b.y) <= tolerance) return { id: line.id, part: 'b' };
    if (distanceToSegment(x, y, a.x, a.y, b.x, b.y) <= tolerance) {
      return { id: line.id, part: 'line' };
    }
  }
  return null;
}
