# 07: Savings — computed total, starting balance, portfolio breakdown

**What to build:** The rest of the Savings page: a computed, ledger-free Savings figure, a one-time starting balance for pre-app savings, and a portfolio percentage breakdown by Holding.

**Blocked by:** 03

**Status:** ready-for-agent

- [x] `GET /api/reports/savings` returns `{ savings_cents, starting_balance_cents, holdings: [{ holding_id, name, type, net_cents, percent }] }`.
- [x] `savings_cents` is cumulative (Income − Expense) since the household started using the app, excluding the Investments category, plus the starting balance.
- [x] The starting balance is stored and edited through the existing `/api/settings` endpoint, extended with a new field.
- [x] The Holdings breakdown is each Holding's net contribution (buys minus sells) as a percentage of the total; a Holding that nets to zero (fully sold) is dropped, not shown at 0%.
- [x] The Savings page shows the computed Savings total, an editable starting balance, and the portfolio breakdown table.
- [x] Backend tests cover the Savings computation and the breakdown math, including a fully-sold Holding dropping out.

## Comments

Implemented `computeSavingsCents(db *sql.DB) (savingsCents, startingBalanceCents int64, err error)` and `readHoldingBreakdown(db *sql.DB) ([]holdingBreakdown, error)` in `cmd/savings.go` — a plain unscoped all-time query (no per-month materialise loop, matching `handleRecentEntries`'s existing precedent for all-time reads) rather than force-fitting the per-month Investments-exclusion helpers.
