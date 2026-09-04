# 02: Daily/weekly expense trend charts

**What to build:** A daily expense trend on the Dashboard and a weekly one on the Month page, both broken down by Category, using shadcn's Chart component. Fully independent of the Investments work — a stock purchase or sale appears in these charts exactly like any other category.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `GET /api/reports/daily` returns a fixed trailing window (e.g. the last 30 days) of per-day, per-category Expense totals, with the same Item-inclusive attribution the existing month breakdown already uses.
- [ ] `GET /api/reports/month/{month}/daily` returns per-day, per-category Expense totals for one calendar month.
- [ ] The Dashboard shows a daily trend chart by Category, sourced from the rolling endpoint.
- [ ] The Month page shows a weekly trend chart by Category for the viewed month, bucketing the daily data into Monday-start weeks on the client; a week that only partially overlaps the month is shown with its real date range rather than hidden or merged.
- [ ] Both charts use the same category names and colors as the rest of the app.
- [ ] shadcn's Chart component is added to the project (`npx shadcn add chart`).
- [ ] Backend tests cover both new report endpoints via the existing `testApp` pattern.
