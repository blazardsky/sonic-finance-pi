# 05: Savings page — savings + investments combined total

**What to build:** The Risparmi card on the Savings page shows a new figure — total Savings plus current portfolio value — below the existing Savings total.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `GET /api/reports/savings` response gains a new field for Savings + portfolio value combined (sum of `SavingsCents` and the holdings' total `net_cents`).
- [ ] Backend test covers the new combined total.
- [ ] The Risparmi card on `Savings.tsx` renders this new total as a second row below the existing Savings figure, clearly labeled as distinct from it.
