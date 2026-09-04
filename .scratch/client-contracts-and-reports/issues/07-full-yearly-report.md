# 07: Full yearly report

**What to build:** A new dedicated report page for a given year: expenses by category for every month, a running total, a total excluding tax, savings at the start of the year, the year's cumulative financial path, and year-scoped medians.

**Blocked by:** `reports-budget-investments/07` (Savings) — only for the "savings at the start of the year" figure, which reuses that ticket's Savings computation with a historical cutoff. Every other part of this report has no dependency on it.

**Status:** ready-for-agent

- [ ] A new report page shows, for a chosen year, expense totals broken down by category for every month, computed as a single query grouped by month and category (not one query per month).
- [ ] A running cumulative total is shown alongside the month-by-month breakdown.
- [ ] The year's total Expense excluding the Taxes category is shown.
- [ ] Savings as it stood on December 31st of the prior year is shown (the existing Savings formula, cut off at that date instead of today).
- [ ] A chart shows the year's cumulative Entrate, Spese, and Net across its months — the "finances path" for the year, built from the same per-month Net figures the existing Year report already computes.
- [ ] Median Expense, Income, and Net are shown, computed over the reported year's own completed months — distinct from Budget's rolling 12-month window.
- [ ] Backend tests cover the grouped query's correctness, the tax-excluded total, and the year-scoped medians.
