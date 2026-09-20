import {
  columnFacetingFeature,
  columnFilteringFeature,
  createFacetedUniqueValues,
  createFilteredRowModel,
  createPaginatedRowModel,
  createSortedRowModel,
  metaHelper,
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
}

// One shared feature set for every DataTable instance in the app (ticket 01:
// core sorting/filtering/pagination/faceting; expansion and grouping are
// layered on top of rendering, not registered here — see tickets 02/03).
export const dataTableFeatures = tableFeatures({
  columnFilteringFeature,
  filteredRowModel: createFilteredRowModel(),
  rowSortingFeature,
  sortedRowModel: createSortedRowModel(),
  rowPaginationFeature,
  paginatedRowModel: createPaginatedRowModel(),
  columnFacetingFeature,
  facetedUniqueValues: createFacetedUniqueValues(),
  columnMeta: metaHelper<DataTableColumnMeta>(),
})

export type DataTableFeatures = typeof dataTableFeatures
