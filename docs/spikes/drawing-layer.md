# Spike: drawing layer on Lightweight Charts v5

| | |
|---|---|
| **Date** | 2026-11-16 |
| **Question** | Does [ADR-0005](../adr/0005-chart-engine.md) hold: our own drawing layer on Lightweight Charts v5, behind the `ChartEngine` adapter? |
| **Branch** | `spike/drawing-layer` (throwaway, not merged) |
| **Library** | `lightweight-charts` 5.3.0 (latest on npm, pinned exactly) |
| **Verdict** | **Holds with changes** (see the end) |

## What was built

| Part | Where | Depends on the library? |
|---|---|---|
| Time model: UTC ms ↔ fractional bar index, across gaps and outside the data | `web/src/chart/model/bars.ts` | No |
| Trend line model, hit-testing, interaction state machine, canvas rendering | `web/src/chart/drawings/` | No |
| `ChartEngine` interface | `web/src/chart/engine/chart-engine.ts` | No |
| Lightweight Charts adapter; overlays render through `attachPrimitive` | `web/src/chart/engine/lwc-engine.ts` | Yes, the only importer (Biome rule) |
| Dev-only route with 10,000 hourly candles, weekdays only | `web/src/app/routes/dev/drawing.tsx` (`/dev/drawing`) | Through the adapter only |

The trend line is created with two clicks, selected by clicking, dragged as a whole or by
either endpoint, and deleted with Delete or Backspace (Escape cancels placing). Anchors are
`(t, p)` in UTC ms and price. Rendering uses series primitives; no DOM overlays were needed.

The model has 27 Vitest tests (time mapping across a weekend gap, before and after the data,
hit-testing order, every interaction path). They run without a canvas.

## Results

Measured in Chrome 157 on macOS (Apple silicon, 10 cores, `devicePixelRatio` 2), driven
through real mouse and keyboard input, except where noted.

| Case | Result |
|---|---|
| Create with two clicks, live preview | Works |
| Drag whole line | Both anchors move by exactly the pointer delta (30.000 px); the chart does not pan |
| Drag one endpoint | The other anchor is unchanged; the chart does not pan |
| Delete / Escape | Works with real key presses; the chart container takes focus on click |
| Hover cursor | `move` on the line, `grab` on endpoints, through the primitive's `hitTest` |
| Wheel zoom, drag pan, log scale, price-axis drag, window resize | Anchors unchanged; our x matches the library's own `timeToCoordinate` to 0 px for every visible bar |
| Endpoint 40 h after the last bar | Drawn at last bar x + 40 × bar spacing; `xToTime` returns exactly +40 h. The library's `timeToCoordinate` returns `null` there |
| Endpoint 60 h before the first bar | Drawn in the empty area at the extrapolated x; round trip exact |
| Time inside a weekend gap (Saturday 23:30) | Drawn halfway between the Friday 23:00 and Monday 00:00 candles |
| x → time → x round trip across the pane | Max error 4e-10 px |
| High DPI (DPR 2) | Canvases are backed at 2×; line and handles render crisp |
| Touch (synthetic `PointerEvent` + `TouchEvent`) | Endpoint drag moves the anchor exactly and does not pan the chart; touching empty space still pans |

Performance while panning one bar per frame for 300 frames (60 Hz display):

| Lines on screen | Frame p50 | Frame p95 | Overlay draw p95 | Hit test per pointer move |
|---|---|---|---|---|
| 0 | 16.7 ms | 18.6 ms | 0.0 ms | — |
| 1 | 16.7 ms | 18.7 ms | 0.1 ms | — |
| 50 | 16.7 ms | 18.6 ms | 0.2 ms | 21 µs |
| 500 | 16.7 ms | 18.6 ms | 0.5 ms | 123 µs |

`performance.now()` is coarsened to 0.1 ms in this browser, so the draw numbers are upper
bounds. 50 lines cost well under 1% of the frame budget.

## What needed workarounds

1. **Fractional bar positions are not supported by the library.** In 5.3.0,
   `timeScale().logicalToCoordinate()` returns **0** for any non-integer index, and
   `coordinateToLogical()` rounds up to a whole index (`indexToCoordinate` checks
   `isInteger`; `coordinateToIndex` uses `Math.ceil`). Without a fix, every anchor between
   bars, which includes any time inside a weekend and any drawing made on a lower timeframe,
   is drawn at x = 0, and every click and drag snaps to whole bars. The mapping is linear in
   the index, so the adapter interpolates between the two neighbouring whole indexes, in both
   directions. This was the only real bug the spike found, and it is invisible in tests that
   anchor only at bar times.
2. **Stopping the chart from panning during a drag.** The library listens to mouse and touch
   events, not pointer events. The drawing layer listens to pointer events in the capture
   phase. When it takes the pointer, `preventDefault()` on `pointerdown` suppresses the
   compatibility mouse events, and `handleScroll`/`handleScale` are switched off until
   `pointerup`, because touch events are not suppressed that way.
3. **Cursor.** The library sets its own cursor on its canvases, so a cursor on the container
   is ignored. The primitive's `hitTest` returning `cursorStyle` is the supported way.
4. **Autoscale.** Price autoscale only considers series data. A drawing can extend above or
   below the visible price range. Primitives can take part through `autoscaleInfo()` if we
   want that. Dragging the price axis turns autoscale off (library behaviour), so the product
   needs a visible way to reset it.

## Not verified

* Real touch devices. Touch was tested with synthetic events in desktop Chrome. Kinetic
  (momentum) scrolling after a flick keeps moving the chart, and its interaction with a touch
  on a drawing needs a device test. A 6 px hit tolerance is also too small for fingers; use a
  larger one for `pointerType === 'touch'`.
* Fractional device pixel ratios (1.25, 1.5 on Windows). Drawing is in media space, so
  axis-aligned lines may blur there; switch those to bitmap space if they do.
* Firefox, Safari and WebKitGTK (the desktop shell, ADR-0009). Performance was measured in
  Chrome only.
* Scroll-back. Anchors before the loaded data are placed by extrapolating with the bar
  interval, so they move to their true position once older bars (with their own gaps) are
  loaded. The stored anchor does not change; only its rendering does.

## Verdict on ADR-0005

**Holds with changes.** Series primitives are enough for a drawing layer, performance has
large headroom, and the drawing model stays independent of the library. The decision itself
does not need to change. Three changes to how it is carried out:

1. **Document the fractional-index workaround** in `10-frontend.md` §4: the adapter must
   interpolate fractional bar positions itself, because the library's coordinate conversions
   are integer-only. Add an E2E or visual test that anchors a line inside a weekend gap, so a
   library upgrade that changes this behaviour is caught (ADR-0005 rule 5 already requires
   upgrades to go through visual regression).
2. **Extend the `ChartEngine` interface** with what drawings need: a way to take the pointer
   (turn pan and zoom off), an overlay that can set the cursor, and fractional
   `timeToX`/`xToTime`. All three are expressed in Wickframe terms, so the boundary still
   holds.
3. **Decide snapping explicitly.** The library snaps to bars by default; the spike removed
   that. Magnet snapping to O/H/L/C (10-frontend §4) should be a feature of the drawing
   layer, not a side effect of the library's rounding.

Before this code moves to `main`: remove the public `chart` field on `LwcEngine` (used only
by the spike's checks), keep the dev route out of production bundles (it currently adds
about 55 KB gzip to the main chunk), and replace the hard-coded drawing colours with design
tokens.
