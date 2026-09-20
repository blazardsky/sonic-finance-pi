# 10: Migrate Holdings to DataTable

**What to build:** `Holdings.tsx` renders through `DataTable` — flat rows, no expansion/grouping needed.

**Blocked by:** 01, 04

**Status:** ready-for-agent

- [ ] Columns: Holding Name (`text`), Type (`select`), Quantity Owned (`range`), Paid (`range`), Actions (no filter, `display` column)
- [ ] Existing per-row dropdown actions (View, Open Pricing, Edit) are preserved in the Actions column
- [ ] Manual check: filter by Type and Quantity range (alone and combined); sort by Paid; confirm existing actions still open the right dialogs; mobile filter sheet parity
