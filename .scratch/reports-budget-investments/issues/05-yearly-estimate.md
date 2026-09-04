# 05: Yearly Estimate

**What to build:** A projected year-end income, expense, and tax figure on the Dashboard, replacing the hardcoded `ESTIMATE_INCOME_CENTS`/`ESTIMATE_EXPENSE_CENTS` targets, computed from this year's pace so far against last year's and always labeled as a projection.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] `GET /api/reports/estimate/{year}` returns `{ year, available, income_estimate_cents, expense_estimate_cents, tax_estimate_cents }`.
- [x] Each estimate is `last year's full-year total × (this year's YTD ÷ last year's YTD at the same point)`, with the YTD cutoff by whole completed months, excluding the Investments category for income/expense.
- [x] A metric is omitted/unavailable when there's no prior year, or last year's YTD-at-the-same-point for that metric is zero (an undefined ratio).
- [x] The Dashboard's progress bars target the computed estimate instead of the fixed constants, and simply don't show when unavailable.
- [x] The Estimate is visually and textually distinguished from actual totals (e.g. a "stima" label) wherever it's shown.
- [x] Backend tests cover the ratio calculation and every unavailable case.

## Comments

Implemented as `cmd/estimate.go` (`GET /api/reports/estimate/{year}`) with `readIncomeCentsExcludingInvestments` added alongside the existing `readExpenseCentsExcludingInvestments`, reusing both in a shared `sumMonths` fold; tax reuses `taxSummary`'s tax-year attribution via a new `readTaxPaidCentsThroughMonths`; `available` is true iff at least one of the three metrics is (which subsumes "no prior year at all" — every metric's ratio is then trivially undefined). Frontend: `Dashboard.tsx` now fetches `EstimateReport` and feeds `income_estimate_cents`/`expense_estimate_cents` into the existing progress bars (already captioned "stimati"), replacing the removed `ESTIMATE_INCOME_CENTS`/`ESTIMATE_EXPENSE_CENTS` constants; a null metric renders no bar via the existing `estimateCents != null` guard. 7 new backend tests in `cmd/estimate_test.go` pin the completed-months cutoff at two points in the year, Investments exclusion on both sides, the no-prior-year case, the per-metric zero-YTD case, and the Tax figure.
