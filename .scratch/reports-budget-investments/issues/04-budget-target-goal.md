# 04: Budget, Target, and Goal

**What to build:** A computed Budget (typical monthly spend) and household-settable Target and savings Goal, shown on the Month page for the current month only.

**Blocked by:** 01

**Status:** ready-for-agent

- [x] `GET /api/reports/budget` returns `{ available, budget_cents, target_cents, goal_cents }`.
- [x] Budget is the median of the trailing 12 *completed* months' Expense totals, excluding the Investments category; `available` is `false` with fewer than 3 completed months of history.
- [x] Target is a household-set value, defaulting to the current Budget the first time it's read if never explicitly set, and stored once set.
- [x] Goal is a household-set monthly savings figure, stored independently of Budget/Target.
- [x] Target and Goal are read/written through the existing `/api/settings` endpoint, extended with the new fields.
- [x] The Month page shows Budget/Target/Goal only when viewing the current real month.
- [x] Backend tests cover the median calculation, the "unavailable" threshold, Target's default-then-sticky behavior, and the Investments exclusion.

## Comments

Implemented `GET /api/reports/budget` (`cmd/budget.go`) plus `readExpenseCentsExcludingInvestments` in `cmd/reports.go` (the shared exclusion helper, reusable by ticket 05's Estimate); "completed months of history" is anchored to the earliest recorded Expense (`firstExpenseMonth`), since the schema has no separate "started using this app on" date. Target/Goal extend the existing `lists` struct/`/api/settings` payload (`cmd/setting.go`) with `target_cents`/`goal_cents`. Month page and Settings screen updated to match.
