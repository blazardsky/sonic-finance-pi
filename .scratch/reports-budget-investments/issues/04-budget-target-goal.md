# 04: Budget, Target, and Goal

**What to build:** A computed Budget (typical monthly spend) and household-settable Target and savings Goal, shown on the Month page for the current month only.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] `GET /api/reports/budget` returns `{ available, budget_cents, target_cents, goal_cents }`.
- [ ] Budget is the median of the trailing 12 *completed* months' Expense totals, excluding the Investments category; `available` is `false` with fewer than 3 completed months of history.
- [ ] Target is a household-set value, defaulting to the current Budget the first time it's read if never explicitly set, and stored once set.
- [ ] Goal is a household-set monthly savings figure, stored independently of Budget/Target.
- [ ] Target and Goal are read/written through the existing `/api/settings` endpoint, extended with the new fields.
- [ ] The Month page shows Budget/Target/Goal only when viewing the current real month.
- [ ] Backend tests cover the median calculation, the "unavailable" threshold, Target's default-then-sticky behavior, and the Investments exclusion.
