# 01: DataTable core — columns, sorting, filtering, pagination

**What to build:** The foundational shared `DataTable` component (`web/src/components/data-table/`) on `@tanstack/react-table`: flat rows only (no expansion/grouping yet — tickets 02/03), sortable column headers, per-column filter widgets driven by a `meta.filterVariant` on each column def, and client-side pagination via a shared `DataTablePagination` piece. This ticket does not touch any page yet — it's the reusable core the later migration tickets build on.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `@tanstack/react-table` (`^9.2.4`) added to `web/package.json`
- [ ] `DataTable<TData>` component accepts `columns: ColumnDef<TData>[]` and `data: TData[]`, renders using the existing shadcn `Table`/`TableHeader`/`TableRow`/`TableCell` primitives (`web/src/components/ui/table.tsx`) via `flexRender`
- [ ] `columnFilters` and `sorting` state live inside `DataTable` itself (`useState`, not lifted to the caller, not persisted to `localStorage`/URL) — this is deliberate, see spec's "Filter/sort state lifetime"
- [ ] A `DataTableColumnHeader` piece renders each non-`display` column's label with a clickable sort toggle (ascending/descending/none) and a sort-direction icon
- [ ] Filter widgets, chosen per column via `column.columnDef.meta?.filterVariant`:
  - `"text"`: `Input`, TanStack's built-in `includesString` filter fn
  - `"select"`: shadcn `Select` populated from `column.getFacetedUniqueValues()` plus a no-filter "all" option
  - `"range"`: two `Input type="number"` (min/max), custom filter fn comparing against `[min, max]` with either bound optional
  - `"date-range"`: two instances of the existing `web/src/components/date-picker.tsx`, same `[min, max]` filter fn shape as `range` but comparing ISO date strings
  - A column with no `filterVariant` renders no filter control
  - Filter controls render inline in the header on desktop; ticket 04 adds the mobile sheet reading/writing this same `columnFilters` state
- [ ] `DataTablePagination`: extracted from the current copy-pasted arrows JSX in `Expenses.tsx`/`Incomes.tsx` (icon-only below `sm`, text+icon above, rendered above and below the table), rewired to `table.previousPage()`/`table.nextPage()`/`table.getCanPreviousPage()`/`table.getCanNextPage()` instead of manual `page` state + length heuristic
- [ ] `DataTable` accepts an optional `initialState.sorting` so a page can set its own default sort order (e.g. Expenses/Incomes default newest-first)
- [ ] Multiple simultaneous column filters combine with AND (TanStack's default) — verify no page needs OR semantics before assuming this
- [ ] No page is migrated in this ticket — first real exercise of `DataTable` is ticket 05 (Expenses)
