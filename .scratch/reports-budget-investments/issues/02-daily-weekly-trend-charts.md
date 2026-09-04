# 02: Daily/weekly expense trend charts

**What to build:** A daily expense trend on the Dashboard and a weekly one on the Month page, both broken down by Category, using shadcn's Chart component. Fully independent of the Investments work — a stock purchase or sale appears in these charts exactly like any other category.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [x] `GET /api/reports/daily` returns a fixed trailing window (e.g. the last 30 days) of per-day, per-category Expense totals, with the same Item-inclusive attribution the existing month breakdown already uses.
- [x] `GET /api/reports/month/{month}/daily` returns per-day, per-category Expense totals for one calendar month.
- [x] The Dashboard shows a daily trend chart by Category, sourced from the rolling endpoint.
- [x] The Month page shows a weekly trend chart by Category for the viewed month, bucketing the daily data into Monday-start weeks on the client; a week that only partially overlaps the month is shown with its real date range rather than hidden or merged.
- [x] Both charts use the same category names and colors as the rest of the app.
- [x] shadcn's Chart component is added to the project (`npx shadcn add chart`).
- [x] Backend tests cover both new report endpoints via the existing `testApp` pattern.

## Comments

Backend: added `readDailyBreakdown` (shared by both endpoints, parameterized by date range) plus `handleDailyReport`/`handleMonthDailyReport` in `cmd/reports.go`, wired in `cmd/app.go`, tested in `cmd/reports_test.go`. Frontend: shadcn's chart component and `recharts` were already installed by a prior commit (Dashboard's net-worth sparkline) — reused rather than reinstalled; added `web/src/lib/trend.ts` (day/week bucketing + category color, keyed by category id off the app's existing `--chart-1..5` tokens) and `web/src/components/CategoryTrendChart.tsx` (shared stacked-bar chart), wired into `Dashboard.tsx` and `Month.tsx`. Categories beyond 5 with spend in the same window will share a color — marked with a `ponytail:` comment in `trend.ts`.
