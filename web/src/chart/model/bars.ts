// SPDX-License-Identifier: AGPL-3.0-only

/** A candle. `t` is the bar open time in UTC ms. */
export interface Bar {
  t: number;
  o: number;
  h: number;
  l: number;
  c: number;
}

/**
 * Fractional bar index of any UTC-ms time. Inside the data it interpolates between neighbouring
 * bars, so times inside session gaps (weekends) land between the two bars around the gap.
 * Before the first bar and after the last it extrapolates by `intervalMs`.
 */
export function timeToLogical(
  bars: readonly { t: number }[],
  timeMs: number,
  intervalMs: number,
): number | null {
  const n = bars.length;
  const first = bars[0];
  const last = bars[n - 1];
  if (!first || !last) return null;
  if (timeMs <= first.t) return (timeMs - first.t) / intervalMs;
  if (timeMs >= last.t) return n - 1 + (timeMs - last.t) / intervalMs;
  let lo = 0;
  let hi = n - 1;
  while (hi - lo > 1) {
    const mid = (lo + hi) >> 1;
    if ((bars[mid] as { t: number }).t <= timeMs) lo = mid;
    else hi = mid;
  }
  const t0 = (bars[lo] as { t: number }).t;
  const t1 = (bars[hi] as { t: number }).t;
  return lo + (timeMs - t0) / (t1 - t0);
}

/** Inverse of timeToLogical. */
export function logicalToTime(
  bars: readonly { t: number }[],
  logical: number,
  intervalMs: number,
): number | null {
  const n = bars.length;
  const first = bars[0];
  const last = bars[n - 1];
  if (!first || !last) return null;
  if (logical <= 0) return first.t + logical * intervalMs;
  if (logical >= n - 1) return last.t + (logical - (n - 1)) * intervalMs;
  const i = Math.floor(logical);
  const t0 = (bars[i] as { t: number }).t;
  const t1 = (bars[i + 1] as { t: number }).t;
  return t0 + (logical - i) * (t1 - t0);
}
