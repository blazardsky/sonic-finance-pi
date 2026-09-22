# 05: Year review line chart

**What to build:** A line chart on the year review showing, per month, what share of that month's classified spend was Necessity, Desire, Wise, and Bullshit — so the household can see how the balance shifts across the year.

**Blocked by:** 03

**Status:** ready-for-agent

- [ ] `GET /api/reports/year/{year}/full` (`fullYearReport`) gains a per-month grouped total — month, necessity_cents, desire_cents, wise_cents, bullshit_cents — in the same shape `ByMonth`/`monthCategoryTotal` already uses
- [ ] A `desire` Expense with no Wise/Bullshit refinement counts toward `desire_cents` only, never `wise_cents` or `bullshit_cents`
- [ ] An Expense with no `spending_intent` set contributes to none of the four buckets
- [ ] Incomes never contribute — Expense/Recurring-only, as scoped in the spec
- [ ] `YearlyReport.tsx` gets a new line chart, behind `spending_intent_enabled`, reusing `FinancesPathChart`'s existing pattern with four series computed client-side as percentages: Necessity/Desire mirror each other around 100% of that month's tagged spend; Wise/Bullshit are computed against that month's Desire spend specifically, not total spend
- [ ] A month with nothing classified at all shows flat/empty lines for that month, not an error or a misleading zero
- [ ] Test: `SpendingIntentByMonth` sums Expense amounts into the correct month/bucket per the rules above
