# 08: Migrate RecurringExpenses to DataTable

**What to build:** `RecurringExpenses.tsx` renders through `DataTable`, using `renderSubRow` (ticket 02) for its existing expandable Details.

**Blocked by:** 01, 02, 04

**Status:** ready-for-agent

- [ ] Columns: Category (`select`), Status (`select` — ongoing/ended, PAC/expense badges), Amount (`range`), Actions (no filter, `display` column) — Note stays inside the expandable Details rather than becoming its own column (it's already hidden on mobile today)
- [ ] `renderSubRow` renders the existing Details content (Note, window, day-of-month, Holding/Subcategory if set) exactly as today
- [ ] Existing per-row dropdown actions (Edit, End, Replace/new amount, Delete) are preserved in the Actions column
- [ ] Manual check: expand/collapse still works and shows the right Details; filter by Category and Status (alone and combined); confirm PAC badge and ongoing/ended badge rendering is unchanged; mobile filter sheet parity
