Status: ready-for-agent

# Real Data Tables (TanStack Table) with Per-Column Filters

## Problem Statement

Every list page in the app renders a plain HTML table (`web/src/components/ui/table.tsx`, a thin shadcn styling wrapper with no behavior of its own). There is no way to filter a list by anything other than the year/period stepper a couple of pages already have, and no way to sort a column — Expenses and Incomes are always newest-first, everything else is in whatever order the backend returns. `v1.3`'s TODO item asks to replace these with real data tables (TanStack Table) with per-column filters.

## Solution

A single shared `DataTable` component, built on `@tanstack/react-table` v9 (the current stable release, and what shadcn's own current data-table guide targets), replaces the plain `<Table>` on every page whose rows are independent records. It provides sortable column headers, an inline filter control per filterable column (desktop), a filter-sheet drawer with the same fields as a stacked form (mobile, since there are no column headers to attach inline filters to), and pagination. Two pages that already page through the backend (Expenses, Incomes) switch to fetching a whole year at once and paginating client-side instead, since both endpoints already support an unpaginated, year-scoped GET with no backend change needed. Three pages need more than flat rows — Clients (expandable Contract sub-rows), RecurringExpenses (expandable Details), and Categories (Spese/Entrate/Entrambi section grouping) — so the shared component supports row expansion and row grouping natively rather than special-casing those pages out of it.

Pivot/grouped displays that aren't row-per-record — YearlyReport's month×category grid, Tracker's rowSpan-merged item/store table — and Dashboard's small preview tables are explicitly left as plain tables; TanStack row semantics don't map onto them.

## User Stories

1. As a household member, I want to filter a list (Expenses, Incomes, Clients, RecurringExpenses, Categories, Subcategories, Holdings, Investments, Savings) by a value in one of its columns, so I can find what I'm looking for without scrolling through everything.
2. As a household member, I want to combine filters on more than one column at once (e.g. category AND store), so I can narrow down to exactly the rows I care about.
3. As a household member, I want to click a column header to sort by it, so I can e.g. see the largest expenses first without waiting for a new report.
4. As a household member using the app on my phone, I want the same filtering ability through a filter sheet, since there's no table header to tap on there.
5. As a household member, I want my active filters to survive switching between years or months on the same page (Expenses/Incomes), so narrowing down to one store or category doesn't reset every time I page back a year.
6. As a household member, I want my filters to start fresh when I come back to a page later, so a filter I forgot about doesn't silently hide rows on my next visit.
7. As a household member, I want Clients' Contract sub-rows, RecurringExpenses' Details, and Categories' section grouping to keep working exactly as they do today, just inside the new table.
8. As a household member, I want Expenses and Incomes to keep their existing "previous/next" page arrows, now paging through a whole year's rows client-side instead of asking the server for the next 50.

## Implementation Decisions

### Dependency

- `@tanstack/react-table` (`^9.2.4`, current stable — not the beta series) added to `web/package.json`. No other new dependency: `Sheet`, `Select`, `Popover`+`Calendar` (via the existing `web/src/components/date-picker.tsx`), `Checkbox`, `DropdownMenu` already exist in `web/src/components/ui/` and cover every filter-widget and interaction need below.

### Shared `DataTable` component

New file(s) under `web/src/components/data-table/` (a `DataTable.tsx` plus `DataTableFilterInput.tsx`/`DataTableMobileFilters.tsx` as the pieces grow — exact split left to the implementing ticket, but everything shared lives in this one directory rather than scattered per-page).

- **Column defs**: each page supplies `ColumnDef<TData>[]` (TanStack's own type) via `columnHelper.accessor`/`columnHelper.display`, same pattern as shadcn's guide. A column opts into filtering by setting `meta: { filterVariant: "text" | "select" | "range" | "date-range" }` — the `DataTable` reads `filterVariant` to decide which control to render, rather than every page hand-rolling its own filter input.
- **Filter widgets by variant**:
  - `text`: a plain `Input`, TanStack's built-in `includesString` filter fn (case-insensitive contains) — for Store, Note, Client Name, Holding Name, Item Name.
  - `select`: a shadcn `Select` populated from the column's distinct values (via `column.getFacetedUniqueValues()`, client-side since all data for the visible page is already loaded) plus an "Tutti/Tutte" no-filter option — for Category, Subcategory, Payer, Applies To, Status/PAC badges.
  - `range`: two `Input type="number"` (min/max) — for Amount columns, using a small custom `filterFn` comparing against `[min, max]` (undefined bound = no limit on that side).
  - `date-range`: two `DatePicker` instances (from/to), same custom-range `filterFn` shape as `range` but comparing ISO date strings — for Date columns.
  - A column with no `filterVariant` in its `meta` gets no filter control at all (e.g. an Actions column, or Clients' expand toggle).
- **Sorting**: every data column (not `display` columns like Actions/expand-toggle) gets a clickable header via a shared `DataTableColumnHeader` (label + sort icon, `column.toggleSorting()`), following shadcn's own recipe. Default sort order per page matches today's (e.g. Expenses/Incomes default to `occurred_on`/`payment_date` descending) via `initialState.sorting`.
- **Pagination**: `rowPaginationFeature` + `createPaginatedRowModel()`. The existing pagination arrows JSX (currently copy-pasted between `Expenses.tsx` and `Incomes.tsx`) is extracted into one shared `DataTablePagination` piece reused by every migrated page (not just those two), wired to `table.previousPage()`/`table.nextPage()`/`table.getCanPreviousPage()`/`table.getCanNextPage()` instead of the current manual `page` state + length heuristic. Same visual design (icon-only below `sm`, text+icon above, rendered above and below the table) — this is a rewire, not a redesign.
- **Row expansion** (Clients, RecurringExpenses): `rowExpandingFeature` + an optional `renderSubRow?: (row: Row<TData>) => React.ReactNode` prop on `DataTable`. When set, an expand-toggle `display` column is auto-injected as the first column, and an expanded row renders `renderSubRow`'s output as an extra `<TableRow>` directly below — same visual shape Clients/RecurringExpenses already have, now driven by TanStack's `row.getIsExpanded()`/`row.toggleExpanded()` instead of local component state.
- **Row grouping** (Categories/Subcategories): a `groupBy?: keyof TData` prop. When set, `DataTable` partitions the already-filtered/sorted rows by that field's value (in a fixed order the page provides, e.g. Spese/Entrate/Entrambi) and renders a section-header `<TableRow>` between groups — the same shape `Categories.tsx`'s `bySection()` already produces, moved inside the shared component so filtering/sorting apply within each section correctly (a filter narrows rows inside a section; it never merges or reorders the sections themselves).
- **Mobile filter sheet**: a `Sheet` (shadcn) triggered by a filter icon button, rendered by `DataTable` itself whenever at least one column has a `filterVariant` — listing every filterable column's control (same widgets as above, e.g. the `select`/`range`/`date-range`/`text` pieces) stacked as a form, reading and writing the exact same `columnFilters` state the desktop inline headers use. There is no separate mobile filter state — mobile and desktop are two views onto one `columnFilters` array.
- **Filter/sort state lifetime**: `columnFilters`/`sorting` live in `DataTable`'s own `useState`, not lifted to the page or persisted to `localStorage`/the URL. This alone satisfies both halves of the persistence story: a page like Expenses/Incomes that changes its fetched `year` without unmounting the `DataTable` (i.e. no `key={year}` on it — the component instance survives, only its `data` prop changes) keeps its filters exactly because React state survives a re-render; navigating away to a different screen and back unmounts and remounts the page (and therefore the `DataTable` instance), which is what resets filters to empty. **Migrating pages must not put a period-derived value in the `DataTable`'s `key`** — that would silently break story 5.

### Expenses/Incomes: drop server-side pagination

- Both `handleListExpenses`/`handleListIncomes` (`cmd/expense.go`, `cmd/income.go`) already support a bare, unpaginated `year`-scoped GET — "a bare GET still returns every Expense unfiltered" / `limit`/`offset` are optional. **No backend change.**
- `Expenses.tsx`/`Incomes.tsx` drop the `limit`/`offset` query params and the `page` state entirely; the existing `year`/"recenti" scoping stays as today. The full year's rows (or, for "recenti", whatever that view already requests) are handed to `DataTable`, which paginates them client-side via `rowPaginationFeature`.

### Per-page migration notes

- **Expenses / Incomes**: columns = Date (`date-range`), Category (`select`), Store/Client (`text`/`select`), Payer (`select`), Amount (`range`), Note (`text`), Actions (no filter). Drop the manual `pageControls` JSX block from both files in favor of the shared `DataTablePagination`.
- **Clients**: `renderSubRow` renders the existing Contracts sub-table content; Category/Total Earned get filter variants (`select`/`range`), Actions stays unfiltered.
- **RecurringExpenses**: `renderSubRow` renders the existing Details content; Category/Status get filter variants.
- **Categories / Subcategories**: `groupBy: "applies_to"` with the fixed Spese/Entrate/Entrambi order; Name gets `text`, Applies To itself is the grouping key so it doesn't also need its own column filter.
- **Holdings**: Name (`text`), Type (`select`), Quantity/Paid (`range`).
- **Investments**: Holding (`select`), kind buy/sell (`select`), Date (`date-range`), Amount (`range`) — table stays read-only (no row click/actions), only sort/filter/pagination are new.
- **Savings**: Holding (`select`), Type (`select`), Amount/% (`range`) — same read-only treatment as Investments.

## Testing Decisions

- No new frontend test runner, consistent with this repo's established practice across every prior spec (e.g. `category-color-and-safe-delete`'s Testing Decisions) — no automated test for `DataTable` itself, its filter widgets, expansion, or grouping.
- Manual check per migrated page: apply a filter on each filterable column (alone and combined with a second one), confirm sorting each column header flips ascending/descending, confirm pagination arrows page correctly through more than one page of results, confirm the mobile filter sheet's fields match the desktop header filters and both stay in sync, and — for Clients/RecurringExpenses/Categories specifically — confirm expansion/grouping still look and behave as they do today.
- Manual check for the persistence story: filter Expenses by category, step to a different year, confirm the category filter is still applied; navigate to another page (e.g. Dashboard) and back to Expenses, confirm the filter has reset.
- Backend: no test changes, since no backend code changes (Expenses/Incomes pagination removal is frontend-only).

## Out of Scope

- YearlyReport's month×category pivot grid and Tracker's rowSpan-grouped item/store table — stay plain `<Table>`, no TanStack.
- Dashboard's preview tables (e.g. "clienti questo mese") — stay plain `<Table>`.
- URL-based or `localStorage`-based filter persistence — filters reset on page remount, per the household's own choice.
- Any backend change — every migrated page's data need is already met by an existing endpoint.
- Global (cross-column) search/filter — only per-column filters are in scope, matching the TODO item as written.
- Column visibility toggling / column reordering / row selection (checkboxes) — none of these were asked for; `DataTable` doesn't wire up `columnVisibilityFeature` or `rowSelectionFeature` even though TanStack offers them.
- Virtualization — none of these lists are large enough (household-scale data on a Pi Zero W) to need it.

## Further Notes

- Builds on the existing `web/src/components/date-picker.tsx` (Popover+Calendar) for `date-range` filters rather than introducing a new date-range picker.
- The pagination-arrow deduplication (`DataTablePagination`) is a side benefit of this migration, not a separate goal — `.scratch/v1.2.4/issues/02-mobile-pagination-arrows.md` is the ticket that originally introduced the copy-pasted version being replaced here.
- This whole spec was shaped by an interactive grilling session in this repo's conversation history rather than a written brief — if anything here reads ambiguous, that transcript is the tie-breaker, not a guess.
