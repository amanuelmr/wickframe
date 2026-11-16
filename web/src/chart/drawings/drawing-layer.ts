// SPDX-License-Identifier: AGPL-3.0-only

import {
  type Anchor,
  type Hit,
  hitTest,
  type Projection,
  project,
  type TrendLine,
} from './trend-line';

interface Drag {
  hit: Hit;
  startX: number;
  startY: number;
  orig: TrendLine;
}

/**
 * Trend-line interaction: two clicks to create, click to select, drag a line or an endpoint,
 * Delete or Backspace to remove. Pointer positions are CSS pixels of the price pane. Pointer
 * handlers return true when they consumed the event, so the chart must not pan or zoom.
 */
export class DrawingLayer {
  lines: TrendLine[] = [];
  selectedId: string | null = null;
  /** First anchor and pointer position while placing a new line. */
  pending: { a: Anchor | null; b: Anchor | null } | null = null;
  private drag: Drag | null = null;

  constructor(
    private readonly proj: Projection,
    private readonly changed: () => void,
    private readonly newId: () => string = () => crypto.randomUUID(),
  ) {}

  startPlacing(): void {
    this.pending = { a: null, b: null };
    this.selectedId = null;
    this.changed();
  }

  get dragging(): boolean {
    return this.drag !== null;
  }

  hover(x: number, y: number): Hit | null {
    return hitTest(this.lines, this.proj, x, y);
  }

  pointerDown(x: number, y: number): boolean {
    if (this.pending) {
      const at = this.anchorAt(x, y);
      if (!at) return true;
      if (!this.pending.a) {
        this.pending.a = at;
      } else {
        const line = { id: this.newId(), a: this.pending.a, b: at };
        this.lines.push(line);
        this.selectedId = line.id;
        this.pending = null;
      }
      this.changed();
      return true;
    }
    const hit = this.hover(x, y);
    const wasSelected = this.selectedId;
    this.selectedId = hit?.id ?? null;
    if (hit) {
      const orig = this.lines.find((l) => l.id === hit.id) as TrendLine;
      this.drag = { hit, startX: x, startY: y, orig: { ...orig } };
    }
    if (hit || wasSelected) this.changed();
    return hit !== null;
  }

  pointerMove(x: number, y: number): boolean {
    if (this.pending?.a) {
      this.pending.b = this.anchorAt(x, y);
      this.changed();
      return true;
    }
    const d = this.drag;
    if (!d) return false;
    const line = this.lines.find((l) => l.id === d.hit.id);
    if (!line) return false;
    if (d.hit.part === 'line') {
      // Move in pixel space and re-anchor, so the line keeps its on-screen shape across gaps.
      const a = this.shifted(d.orig.a, x - d.startX, y - d.startY);
      const b = this.shifted(d.orig.b, x - d.startX, y - d.startY);
      if (a && b) {
        line.a = a;
        line.b = b;
      }
    } else {
      const at = this.anchorAt(x, y);
      if (at) line[d.hit.part] = at;
    }
    this.changed();
    return true;
  }

  pointerUp(): void {
    this.drag = null;
  }

  /** Returns true when the key was handled. */
  keyDown(key: string): boolean {
    if (key === 'Escape' && this.pending) {
      this.pending = null;
    } else if ((key === 'Delete' || key === 'Backspace') && this.selectedId) {
      this.lines = this.lines.filter((l) => l.id !== this.selectedId);
      this.selectedId = null;
    } else {
      return false;
    }
    this.changed();
    return true;
  }

  private anchorAt(x: number, y: number): Anchor | null {
    const t = this.proj.xToTime(x);
    const p = this.proj.yToPrice(y);
    return t === null || p === null ? null : { t, p };
  }

  private shifted(a: Anchor, dx: number, dy: number): Anchor | null {
    const at = project(this.proj, a);
    return at && this.anchorAt(at.x + dx, at.y + dy);
  }
}
