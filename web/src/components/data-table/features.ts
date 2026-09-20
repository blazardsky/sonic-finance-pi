import {
  columnFacetingFeature,
  columnFilteringFeature,
  createExpandedRowModel,
  createFacetedUniqueValues,
  createFilteredRowModel,
  createPaginatedRowModel,
  createSortedRowModel,
  metaHelper,
  rowExpandingFeature,
  rowPaginationFeature,
  rowSortingFeature,
  tableFeatures,
} from "@tanstack/react-table"

// The widget DataTableFilterInput renders for a column, chosen by whoever
// defines that column's ColumnDef via `meta: { filterVariant: ... }` — a
// column with no filterVariant gets no filter control at all. "range" and
// "date-range" both filter on a [min, max] tuple (TanStack's own
// inNumberRange/inDateRange), one numeric, one ISO-date-string.
export type FilterVariant = "text" | "select" | "range" | "date-range"

export interface DataTableColumnMeta {
  filterVariant?: FilterVariant
  // Matches the `hidden md:table-cell` treatment several pages already gave
  // a column before migrating (RecurringExpenses' Note, hidden on mobile
  // since its content is repeated in the expand panel there) — applied to
  // both the header and body cell.
  hiddenOnMobile?: boolean
}

// One shared feature set for every DataTable instance in the app (ticket 01:
// core sorting/filtering/pagination/faceting; ticket 02 adds expansion.
// Grouping (ticket 03) is plain partitioning done in DataTable's own render,
// not a TanStack feature — TanStack's built-in grouping is an aggregation/
// pivot feature, the wrong semantics for "sort rows into named sections").
export const dataTableFeatures = tableFeatures({
  columnFilteringFeature,
  filteredRowModel: createFilteredRowModel(),
  rowSortingFeature,
  sortedRowModel: createSortedRowModel(),
  rowPaginationFeature,
  paginatedRowModel: createPaginatedRowModel(),
  columnFacetingFeature,
  facetedUniqueValues: createFacetedUniqueValues(),
  rowExpandingFeature,
  expandedRowModel: createExpandedRowModel(),
  columnMeta: metaHelper<DataTableColumnMeta>(),
})

export type DataTableFeatures = typeof dataTableFeatures
