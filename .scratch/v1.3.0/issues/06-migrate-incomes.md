# 06: Migrate Incomes to DataTable

**What to build:** The same treatment as ticket 05 (Expenses), applied to `Incomes.tsx` and `/api/incomes`.

**Blocked by:** 01, 04

**Status:** ready-for-agent

- [ ] `Incomes.tsx` stops sending `limit`/`offset` query params; keeps the existing `year`/"recenti" scoping exactly as today
- [ ] Manual `page` state and the copy-pasted `pageControls` JSX block are removed; `DataTablePagination` (ticket 01) takes over
- [ ] Columns match today's exactly — Date (`date-range`, using payment date), Amount (`range`), Category/income reason (`select`), Actions (no filter, `display` column). Client/Payer/Note are **not** table columns today and this ticket does not add them — same reasoning as ticket 05
- [ ] Default sort: payment date descending, matching today's order, via `initialState.sorting`
- [ ] The `DataTable` instance is not re-mounted when `year` changes — only its `data` prop changes, same reasoning as ticket 05
- [ ] Manual check: same battery as ticket 05 — combined filters, sort both directions, pagination across pages, filters surviving a year switch, filters resetting on navigate-away-and-back, mobile filter sheet parity
