# 07: Full yearly report

**What to build:** A new dedicated report page for a given year: expenses by category for every month, a running total, a total excluding tax, savings at the start of the year, the year's cumulative financial path, and year-scoped medians.

**Blocked by:** `reports-budget-investments/07` (Savings) — only for the "savings at the start of the year" figure, which reuses that ticket's Savings computation with a historical cutoff. Every other part of this report has no dependency on it.

**Status:** ready-for-agent

- [x] A new report page shows, for a chosen year, expense totals broken down by category for every month, computed as a single query grouped by month and category (not one query per month).
- [x] A running cumulative total is shown alongside the month-by-month breakdown.
- [x] The year's total Expense excluding the Taxes category is shown.
- [x] Savings as it stood on December 31st of the prior year is shown (the existing Savings formula, cut off at that date instead of today).
- [x] A chart shows the year's cumulative Entrate, Spese, and Net across its months — the "finances path" for the year, built from the same per-month Net figures the existing Year report already computes.
- [x] Median Expense, Income, and Net are shown, computed over the reported year's own completed months — distinct from Budget's rolling 12-month window.
- [x] Backend tests cover the grouped query's correctness, the tax-excluded total, and the year-scoped medians.

## Comments

Built as `GET /api/reports/year/{year}/full` (cmd/yearreport.go) plus `web/src/pages/YearlyReport.tsx`; `computeSavingsCents` (cmd/savings.go) grew an optional `cutoff` date parameter rather than a duplicate function, since the historical and all-time reads are the same formula at different points in time; the running total and cumulative-path chart are computed client-side from data the backend already returns, per the ticket's own framing that these are rendering additions, not new computations.
