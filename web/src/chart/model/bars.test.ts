// SPDX-License-Identifier: AGPL-3.0-only

import { describe, expect, test } from 'vitest';
import { logicalToTime, timeToLogical } from './bars';

const H = 3_600_000;
// Friday 22:00 and 23:00, then Monday 00:00 and 01:00 (a weekend gap after index 1).
const fri = Date.UTC(2026, 0, 2, 22);
const mon = Date.UTC(2026, 0, 5, 0);
const bars = [{ t: fri }, { t: fri + H }, { t: mon }, { t: mon + H }];

describe('timeToLogical', () => {
  test.each([
    ['an exact bar time', mon, 2],
    ['halfway between two bars', fri + H / 2, 0.5],
    ['inside the weekend gap', fri + H + (mon - fri - H) / 2, 1.5],
    ['before the first bar', fri - 3 * H, -3],
    ['after the last bar, in the future', mon + 11 * H, 13],
  ])('maps %s', (_, t, want) => {
    expect(timeToLogical(bars, t, H)).toBeCloseTo(want, 9);
  });

  test('returns null without bars', () => {
    expect(timeToLogical([], mon, H)).toBeNull();
  });
});

test.each([-3, 0, 0.5, 1.5, 2, 3, 13])('logicalToTime inverts timeToLogical at %d', (logical) => {
  const t = logicalToTime(bars, logical, H) as number;
  expect(timeToLogical(bars, t, H)).toBeCloseTo(logical, 9);
});
