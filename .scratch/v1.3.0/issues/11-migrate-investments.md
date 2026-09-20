# 11: Migrate Investments to DataTable

**What to build:** `Investments.tsx` (the merged buy/sell transaction log) renders through `DataTable` — flat, read-only rows (no row click/actions today, and this migration doesn't add any).

**Blocked by:** 01, 04

**Status:** ready-for-agent

- [ ] Columns match today's exactly — Holding (`select`; the cell text is "Holding name · buy/sell", filter matches against the holding name, kind stays embedded display text rather than becoming a second column), Date (`date-range`), Amount (`range`) — no Actions column, since rows aren't editable here today (edits happen via Expenses/Incomes)
- [ ] Default sort: Date descending, matching today's merged/sorted order
- [ ] Manual check: filter by Holding and kind (alone and combined); sort by Amount; confirm the buy/sell merge from Expenses+Incomes is unaffected by the table swap; mobile filter sheet parity
