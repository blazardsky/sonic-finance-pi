# 03: Annual report — radial & pie charts

**What to build:** Additional radial and/or pie chart(s) on the Report annuale page (`YearlyReport.tsx`), visualizing data it already fetches (the per-category breakdown from `/api/reports/year/{year}/full` is the natural candidate).

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] At least one radial chart and one pie chart added to `YearlyReport.tsx`.
- [x] Charts are built from data the page already receives — no new endpoint unless the existing response shape genuinely can't support it.
- [x] Charts use the same shadcn chart primitives/pattern as `Year.tsx`'s existing `BarChart` (`ChartContainer`/`ChartTooltip`/`ChartLegend`) for visual consistency with the rest of the app.
- [x] The existing full-year-breakdown table and cumulative line chart are unaffected.
