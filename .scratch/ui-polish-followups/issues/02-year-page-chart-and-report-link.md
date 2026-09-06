# 02: Year page — bigger monthly chart & Report annuale moved to a button

**What to build:** On the Year page, give the "Mese per mese" chart a y-axis and more room by stacking the totals cards vertically; separately, move the "Report annuale" link out of the main sidebar into a button at the top of this page. The two report pages/routes/endpoints themselves stay exactly as separate as ADR-0011 decided — this ticket only touches layout and navigation.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] The "Mese per mese" `BarChart` (`Year.tsx`) has a visible y-axis showing expense cent values (it currently has none — only an `XAxis`).
- [ ] The income/expense/difference totals card and the tax-summary card are stacked vertically (one above the other) instead of side-by-side, freeing horizontal width for the chart to render larger.
- [ ] The "Report annuale" (`yearReport`) entry is removed from `app-sidebar.tsx`'s nav group.
- [ ] A button at the top of `Year.tsx` navigates to the Report annuale screen (`yearReport`).
- [ ] `YearlyReport.tsx`, `handleFullYearReport`, and `handleYearReport` are otherwise untouched — no merging of the two pages or their endpoints (ADR-0011 stands).
