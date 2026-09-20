# 07: Migrate Clients to DataTable

**What to build:** `Clients.tsx` renders through `DataTable`, using `renderSubRow` (ticket 02) for its existing expandable Contracts sub-rows.

**Blocked by:** 01, 02, 04

**Status:** ready-for-agent

- [ ] Columns: Client Name (`text`), Default Category (`select`), Total Earned (`range`), Actions (no filter, `display` column) — the expand toggle is auto-injected by `renderSubRow`, not a manual column
- [ ] `renderSubRow` renders the existing Contracts sub-table content exactly as today (same fields, same per-contract actions)
- [ ] Existing per-row dropdown actions (Edit, Add Contract, Hide/Unhide, Delete) are preserved in the Actions column
- [ ] Manual check: expand/collapse still works and shows the right Contracts; filter by Client Name and Default Category (alone and combined); sort by Total Earned; confirm a hidden Client still behaves as it does today (visibility toggle unaffected by the table swap); mobile filter sheet parity
