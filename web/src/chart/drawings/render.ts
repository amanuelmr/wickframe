// SPDX-License-Identifier: AGPL-3.0-only

import type { DrawingLayer } from './drawing-layer';
import { type Anchor, type Projection, project } from './trend-line';

const LINE = '#2962ff';
const HANDLE_RADIUS = 4;

/** Draws every trend line in CSS-pixel space; the caller has already scaled `ctx` for DPR. */
export function drawTrendLines(
  ctx: CanvasRenderingContext2D,
  proj: Projection,
  layer: DrawingLayer,
): void {
  ctx.lineWidth = 2;
  ctx.strokeStyle = LINE;
  ctx.fillStyle = '#ffffff';
  for (const line of layer.lines) {
    const selected = line.id === layer.selectedId;
    segment(ctx, proj, line.a, line.b, selected);
  }
  const pending = layer.pending;
  if (pending?.a) segment(ctx, proj, pending.a, pending.b ?? pending.a, true);
}

function segment(
  ctx: CanvasRenderingContext2D,
  proj: Projection,
  from: Anchor,
  to: Anchor,
  handles: boolean,
): void {
  const a = project(proj, from);
  const b = project(proj, to);
  if (!a || !b) return;
  ctx.beginPath();
  ctx.moveTo(a.x, a.y);
  ctx.lineTo(b.x, b.y);
  ctx.stroke();
  if (!handles) return;
  for (const p of [a, b]) {
    ctx.beginPath();
    ctx.arc(p.x, p.y, HANDLE_RADIUS, 0, 2 * Math.PI);
    ctx.fill();
    ctx.stroke();
  }
}
