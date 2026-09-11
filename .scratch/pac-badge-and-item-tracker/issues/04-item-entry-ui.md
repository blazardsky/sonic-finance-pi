# 04: Item entry UI: quantity/unit/discounted + typeahead

**What to build:** Entering an Item on an Expense lets the household optionally record quantity, unit, and whether it was discounted, sees the computed price per unit inline, and gets live suggestions while typing the Item name or Store.

**Blocked by:** 01, 03

**Status:** ready-for-agent

- [ ] The Expense/Item entry form gains quantity, unit (kg/lt/piece), and discounted inputs per Item, all optional
- [ ] The computed price per unit is displayed inline, read-only, updating as quantity/amount change
- [ ] The Item name field shows live suggestions from ticket 03's endpoint as the household types
- [ ] The Store field shows live suggestions from ticket 03's endpoint as the household types
- [ ] `npm run typecheck` and `npm run build` pass
