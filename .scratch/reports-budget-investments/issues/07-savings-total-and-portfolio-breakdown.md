# 07: Savings — computed total, starting balance, portfolio breakdown

**What to build:** The rest of the Savings page: a computed, ledger-free Savings figure, a one-time starting balance for pre-app savings, and a portfolio percentage breakdown by Holding.

**Blocked by:** 03

**Status:** ready-for-agent

- [ ] `GET /api/reports/savings` returns `{ savings_cents, starting_balance_cents, holdings: [{ holding_id, name, type, net_cents, percent }] }`.
- [ ] `savings_cents` is cumulative (Income − Expense) since the household started using the app, excluding the Investments category, plus the starting balance.
- [ ] The starting balance is stored and edited through the existing `/api/settings` endpoint, extended with a new field.
- [ ] The Holdings breakdown is each Holding's net contribution (buys minus sells) as a percentage of the total; a Holding that nets to zero (fully sold) is dropped, not shown at 0%.
- [ ] The Savings page shows the computed Savings total, an editable starting balance, and the portfolio breakdown table.
- [ ] Backend tests cover the Savings computation and the breakdown math, including a fully-sold Holding dropping out.
