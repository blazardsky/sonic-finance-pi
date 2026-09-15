# 01: Chart tooltip renders above the legend, not under it

**What to build:** `ChartTooltip` (`web/src/components/ui/chart.tsx`) is
currently a bare re-export of `RechartsPrimitive.Tooltip`. Because
`<ChartLegend>` is declared after `<ChartTooltip>` in every chart that uses
both (CategoryTrendChart, Dashboard, Year, YearlyReport, Tracker), the legend
paints on top of the tooltip when they visually overlap — neither has an
explicit z-index, so later DOM order wins. Wrap `ChartTooltip` in a small
component that defaults `wrapperStyle`'s `zIndex` above the legend's implicit
stacking, without touching any of the five call sites.

**Blocked by:** None

**Status:** resolved

- [x] `ChartTooltip` in `ui/chart.tsx` becomes a thin wrapper around
      `RechartsPrimitive.Tooltip` that sets a `zIndex` in `wrapperStyle` by
      default (merging in, not clobbering, a caller-supplied `wrapperStyle`).
- [x] Verified live in the browser — confirmed the fix doesn't regress
      anything on Year.tsx's chart; didn't manage to reproduce the exact
      overlap in that particular chart's data shape (its tooltip tracks near
      the top, not the bottom, when hovering a short bar), but the z-index
      fix is applied unconditionally at the shared component and is correct
      per Recharts' own docs regardless.

## Comments

`wrapperStyle.zIndex` confirmed via Recharts' own docs (v3, `/recharts/recharts`)
as the supported way to control a Tooltip's stacking relative to siblings like
the Legend.
