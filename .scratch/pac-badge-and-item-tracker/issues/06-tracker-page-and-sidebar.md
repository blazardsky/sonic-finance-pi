# 06: Tracker page + sidebar

**What to build:** A new "Tracker" section of the app where the household can browse how item prices have moved over time and across stores.

**Blocked by:** 05

**Status:** ready-for-agent

- [ ] A new fourth sidebar group, "Tracker," alongside the existing Overview/Entries/Management groups
- [ ] A new page rendering a grouped table (item → category → store, showing last/min/max price per store)
- [ ] One line chart per item showing yearly average price, on the same page
- [ ] Sourced entirely from ticket 05's aggregation endpoint
- [ ] `npm run typecheck` and `npm run build` pass
