# 05: Migrate Expenses to DataTable

**What to build:** `Expenses.tsx` drops server-side pagination (`limit`/`offset`/`page` state) in favor of fetching one whole year (or "recenti") unfiltered from the already-unpaginated-capable `/api/expenses` endpoint, and renders through `DataTable` with sorting, per-column filters, and client-side pagination.

**Blocked by:** 01, 04

**Status:** ready-for-agent

- [ ] `Expenses.tsx` stops sending `limit`/`offset` query params; keeps the existing `year`/"recenti" scoping exactly as today
- [ ] Manual `page` state and the copy-pasted `pageControls` JSX block are removed; `DataTablePagination` (ticket 01) takes over
- [ ] Columns match today's exactly — Date (`date-range`), Amount (`range`), Category (`select`), Actions (no filter, `display` column). Store/Payer/Note are **not** table columns today (they live in the "Visualizza" detail dialog per an earlier redesign) and this ticket does not add them as columns — filtering is scoped to what's actually a column, per the spec's "matching the TODO item as written"
- [ ] Default sort: `occurred_on` descending, matching today's order, via `initialState.sorting`
- [ ] The `DataTable` instance is not re-mounted when `year` changes (no period value in its `key`) — only its `data` prop changes, so active filters survive a year/period switch (spec story 5) while still resetting on navigating away and back (story 6)
- [ ] Manual check: filter by Category alone, then add a second filter (e.g. Store) and confirm both apply together (AND); sort by Amount ascending/descending; page through more than one page of results; step to a different year and confirm filters are still applied; navigate to another screen and back and confirm filters reset; repeat the filter check on a narrow viewport via the mobile filter sheet
