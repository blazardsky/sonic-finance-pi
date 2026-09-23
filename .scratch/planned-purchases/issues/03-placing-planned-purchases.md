# 03: Placing Planned purchases

**What to build:** Each Planned purchase shows the month it fits in. The app walks the list in priority order: a purchase lands in the first of the 6 months whose running Headroom (after the purchases above it) covers it, and uses up that amount. One that fits in no month is skipped without blocking the ones after it, and shows "non rientra nei prossimi 6 mesi", how much is missing, and whether Savings cover that gap ("i risparmi coprono la differenza") or a loan would be needed ("servirebbe un finanziamento"). The month rows list the purchases landing in each month.

**Blocked by:** 01, 02

**Status:** ready-for-agent

- [x] Each Planned purchase shows its month, or the "non rientra" state with the missing amount and the Savings answer
- [x] Each month row lists the Planned purchases that land in it
- [x] Reordering the list re-places the purchases immediately
- [x] With the projection unavailable (under 3 months of history), no purchase shows a month
- [x] Tests: priority order decides who gets Headroom first; a purchase that doesn't fit consumes nothing and a smaller later one still lands; missing amount = amount − balance at the end of the horizon after higher-priority placements; Savings cover when Savings ≥ missing amount
