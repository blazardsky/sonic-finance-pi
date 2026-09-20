# 12: Migrate Savings portfolio table to DataTable

**What to build:** The portfolio table on `Savings.tsx` renders through `DataTable` — flat, read-only rows, same treatment as Investments (ticket 11).

**Blocked by:** 01, 04

**Status:** ready-for-agent

- [ ] Columns: Holding (`select`), Type (`select`), Amount (`range`), % (`range`) — no Actions column, this is a derived read-only recap
- [ ] Manual check: filter by Holding and Type (alone and combined); sort by Amount and %; confirm the portfolio figures are unaffected by the table swap; mobile filter sheet parity
