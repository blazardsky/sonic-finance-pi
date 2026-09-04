# 05: Yearly Estimate

**What to build:** A projected year-end income, expense, and tax figure on the Dashboard, replacing the hardcoded `ESTIMATE_INCOME_CENTS`/`ESTIMATE_EXPENSE_CENTS` targets, computed from this year's pace so far against last year's and always labeled as a projection.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `GET /api/reports/estimate/{year}` returns `{ year, available, income_estimate_cents, expense_estimate_cents, tax_estimate_cents }`.
- [ ] Each estimate is `last year's full-year total × (this year's YTD ÷ last year's YTD at the same point)`, with the YTD cutoff by whole completed months, excluding the Investments category for income/expense.
- [ ] A metric is omitted/unavailable when there's no prior year, or last year's YTD-at-the-same-point for that metric is zero (an undefined ratio).
- [ ] The Dashboard's progress bars target the computed estimate instead of the fixed constants, and simply don't show when unavailable.
- [ ] The Estimate is visually and textually distinguished from actual totals (e.g. a "stima" label) wherever it's shown.
- [ ] Backend tests cover the ratio calculation and every unavailable case.
