# 06: Year review diagram

**What to build:** A soft, "by feel" blurred diagram on the year review — three overlapping regions (Necessity, Desire+Wise, Desire+Bullshit) sized roughly by the year's € total in each — for an at-a-glance read of the year's spending character. Desktop only.

**Blocked by:** 05

**Status:** ready-for-agent

- [ ] New diagram component on `YearlyReport.tsx`, behind `spending_intent_enabled`, reusing ticket 05's `SpendingIntentByMonth` data (summed across the year) — no new API endpoint or query
- [ ] Three regions: Necessity, Desire+Wise (`desire_wise`), Desire+Bullshit (`desire_bullshit`) — plain `desire` with no refinement contributes to neither of the latter two
- [ ] Regions are sized only by what's actually classified — an untagged Expense contributes to none of the three
- [ ] Not a mathematically exact Venn/Euler layout — soft/blurred, sized roughly by proportion, per the spec's "by feel" framing
- [ ] Diagram is hidden below the desktop breakpoint, with no alternate mobile view
- [ ] Test (backend, if any new aggregation is added beyond reusing 05's field): totals per region sum correctly from `SpendingIntentByMonth`
