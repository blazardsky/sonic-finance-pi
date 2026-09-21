/* eslint-disable react-refresh/only-export-components */
import * as React from "react"
import { RiArrowDownSLine, RiArrowRightSLine, RiFilter3Line } from "@remixicon/react"
import {
  createColumnHelper,
  useTable,
  type ColumnDef,
  type ColumnFiltersState,
  type ExpandedState,
  type PaginationState,
  type Row,
  type RowData,
  type SortingState,
  type Updater,
} from "@tanstack/react-table"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { t } from "@/lib/strings"
import { cn } from "@/lib/utils"
import {
  dataTableFeatures,
  type DataTableFeatures,
  type FilterVariant,
} from "./features"
import { DataTableColumnHeader } from "./DataTableColumnHeader"
import { DataTableFilterInput } from "./DataTableFilterInput"
import { DataTableMobileFilters } from "./DataTableMobileFilters"
import { DataTablePagination } from "./DataTablePagination"

// Every column ticket 01 defines is a data column; this one is injected by
// DataTable itself when `renderSubRow` is set, always first, never supplied
// by a page.
const EXPAND_COLUMN_ID = "__expand__"

// A column's meta.filterVariant picks the *widget* (DataTableFilterInput);
// this picks the matching *filterFn*, so a page never has to remember to set
// one by hand. TanStack's own "auto" resolution (used when filterFn is
// omitted) instead infers from the column's raw cell value type — which
// happens to land on the right built-in for "range"/"date-range" (numbers
// and date strings resolve to inNumberRange/inDateRange), but gets "select"
// wrong: a string value auto-resolves to includesString (substring match),
// not the exact match a dropdown implies. Explicit beats implicit here.
const FILTER_FN_BY_VARIANT = {
  text: "includesString",
  select: "equalsString",
  range: "inNumberRange",
  "date-range": "inDateRange",
} as const satisfies Record<FilterVariant, string>

// TValue defaults to `any` (not `unknown`) so an array of column defs can mix
// differently-typed columns (a string-accessor column next to a number one)
// — the same reason v8's ColumnDef<TData, any> convention exists.
export type DataTableColumnDef<TData extends RowData> = ColumnDef<
  DataTableFeatures,
  TData,
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- see comment above
  any
>

// A page defines its columns against this rather than raw createColumnHelper,
// so every migrated page gets the same feature set without repeating the
// generic — `createDataTableColumnHelper<Expense>()` in place of
// `createColumnHelper<typeof dataTableFeatures, Expense>()`.
export function createDataTableColumnHelper<TData extends RowData>() {
  return createColumnHelper<DataTableFeatures, TData>()
}

export const DEFAULT_PAGE_SIZE = 25

// The shared table every migrated list page renders through: sortable
// headers, a per-column filter widget for any column carrying
// `meta.filterVariant`, and client-side pagination. Filter/sort/pagination
// state lives here, in plain useState — never lifted to the caller, never
// persisted to localStorage or the URL. That is deliberate (spec's "Filter/
// sort state lifetime"): a page that changes its `data` prop without
// remounting this component (e.g. stepping to a different year) keeps
// whatever filters were active, because the state simply survives the
// re-render; navigating away to a different screen and back unmounts and
// remounts this component, which is what resets them. A page must not put a
// period-derived value in this component's own React `key` — that would
// silently defeat the persistence story.
export function DataTable<TData extends RowData>({
  columns,
  data,
  initialSorting,
  pageSize = DEFAULT_PAGE_SIZE,
  getRowId,
  paginationPosition = "bottom",
  renderSubRow,
  groupBy,
  pagination: controlledPagination,
  onPaginationChange: controlledOnPaginationChange,
}: {
  columns: DataTableColumnDef<TData>[]
  data: TData[]
  initialSorting?: SortingState
  pageSize?: number
  getRowId?: (row: TData, index: number) => string
  paginationPosition?: "top" | "bottom" | "both"
  // Optionally-controlled, same shape as e.g. a Dialog's open/onOpenChange:
  // omit both and DataTable manages pageIndex/pageSize itself (the default,
  // and what every page but Investments uses); pass both to source them from
  // outside instead — Investments does this to keep pagination in the URL,
  // surviving a reload/back-forward, not just a data refetch.
  pagination?: PaginationState
  onPaginationChange?: (updater: Updater<PaginationState>) => void
  // Ticket 02: when set, an expand-toggle column is auto-injected as the
  // first column and an expanded row renders this directly beneath it — the
  // "detail panel" pattern (Clients' Contracts, RecurringExpenses' Details),
  // not TanStack's own tree-of-same-shaped-rows expansion.
  renderSubRow?: (row: Row<DataTableFeatures, TData>) => React.ReactNode
  // Ticket 03: partitions the filtered/sorted rows into named sections in a
  // fixed order (Categories' Spese/Entrate/Entrambi) — plain partitioning in
  // this component's own render, not TanStack's grouping feature (that one
  // aggregates/pivots, the wrong semantics here). An empty section renders no
  // header at all, matching Categories.tsx's own `bySection` today. Disables
  // pagination: grouped lists in this app are small, fixed-size, and slicing
  // a page out of a grouped set would split or hide whole sections.
  groupBy?: {
    key: keyof TData
    order: readonly TData[keyof TData][]
    label?: (value: TData[keyof TData]) => React.ReactNode
  }
}) {
  const [sorting, setSorting] = React.useState<SortingState>(initialSorting ?? [])
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([])
  const [internalPagination, setInternalPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize,
  })
  const pagination = controlledPagination ?? internalPagination
  const setPagination = controlledOnPaginationChange ?? setInternalPagination
  const [expanded, setExpanded] = React.useState<ExpandedState>({})
  // Desktop's inline header filters start hidden, revealed by the toggle
  // button below — mobile's own Sheet (DataTableMobileFilters) already
  // starts closed the same way, via its own independent open state; the two
  // aren't the same boolean (a shared one would pop the Sheet open on
  // desktop too, since Radix doesn't know about breakpoints), just the same
  // closed-by-default, click-to-reveal pattern.
  const [filtersOpen, setFiltersOpen] = React.useState(false)

  const tableColumns = React.useMemo(() => {
    // Every filterable column gets its variant's matching filterFn, unless
    // the page already set its own explicitly.
    const withFilterFns = columns.map((column) => {
      const variant = column.meta?.filterVariant
      if (!variant || column.filterFn) return column
      return {
        ...column,
        filterFn: FILTER_FN_BY_VARIANT[variant],
      } as unknown as DataTableColumnDef<TData>
    })

    if (!renderSubRow) return withFilterFns
    const expandColumn: DataTableColumnDef<TData> = {
      id: EXPAND_COLUMN_ID,
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t.details}
          aria-expanded={row.getIsExpanded()}
          onClick={() => row.toggleExpanded()}
        >
          {row.getIsExpanded() ? <RiArrowDownSLine /> : <RiArrowRightSLine />}
        </Button>
      ),
    }
    return [expandColumn, ...withFilterFns]
  }, [columns, renderSubRow])

  const table = useTable({
    features: dataTableFeatures,
    columns: tableColumns,
    data,
    getRowId,
    getRowCanExpand: renderSubRow ? () => true : undefined,
    state: { sorting, columnFilters, pagination, expanded },
    // TanStack's own default resets pageIndex to 0 whenever the row model
    // recomputes for *any* reason, including `data` simply getting a new
    // array reference from a reload after an unrelated save — which is what
    // was silently bouncing every page back to page 1 after every edit.
    // Filtering still explicitly resets the page itself, below, since a
    // narrower result set landing on a now-out-of-range page is a real
    // problem this doesn't otherwise guard against.
    autoResetPageIndex: false,
    onSortingChange: setSorting,
    // Resets to the first page whenever the active filters change — a filter
    // narrowing the result set to fewer than the current page index's worth
    // of rows would otherwise land on an empty page.
    onColumnFiltersChange: (updater) => {
      setColumnFilters(updater)
      setPagination((p) => ({ ...p, pageIndex: 0 }))
    },
    onPaginationChange: setPagination,
    onExpandedChange: setExpanded,
  })

  const leafColumnCount = table.getAllLeafColumns().length
  const pager = <DataTablePagination table={table} />

  const filterableColumns = table
    .getAllLeafColumns()
    .filter((column) => column.columnDef.meta?.filterVariant)
  const activeFilterCount = filterableColumns.filter(
    (column) => column.getFilterValue() !== undefined
  ).length

  const filterButton = filterableColumns.length > 0 && (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className="hidden w-fit gap-1.5 sm:inline-flex"
      aria-pressed={filtersOpen}
      onClick={() => setFiltersOpen((v) => !v)}
    >
      <RiFilter3Line className="size-4" />
      {t.dataTableFilters}
      {activeFilterCount > 0 && (
        <Badge variant="secondary" className="px-1.5">
          {activeFilterCount}
        </Badge>
      )}
    </Button>
  )
  // Filtri sits next to the prev/next buttons, the whole row pushed to the
  // right — not a separate row of its own above the table.
  const pagerRow = (
    <div className="flex items-center justify-end gap-2">
      {filterButton}
      {pager}
    </div>
  )

  const renderDataRow = (row: Row<DataTableFeatures, TData>) => (
    <React.Fragment key={row.id}>
      <TableRow>
        {row.getAllCells().map((cell) => (
          <TableCell
            key={cell.id}
            className={cn(
              cell.column.columnDef.meta?.hiddenOnMobile && "hidden md:table-cell"
            )}
          >
            <table.FlexRender cell={cell} />
          </TableCell>
        ))}
      </TableRow>
      {renderSubRow && row.getIsExpanded() && (
        <TableRow>
          <TableCell colSpan={leafColumnCount}>{renderSubRow(row)}</TableCell>
        </TableRow>
      )}
    </React.Fragment>
  )

  // Grouped tables read the sorted-but-not-yet-paginated row model and skip
  // pagination entirely (see the groupBy prop's own doc comment above).
  const rows = groupBy ? table.getSortedRowModel().rows : table.getRowModel().rows

  return (
    <div className="flex flex-col gap-2">
      <DataTableMobileFilters table={table} />
      {groupBy && filterButton && (
        <div className="flex justify-end">{filterButton}</div>
      )}
      {!groupBy &&
        (paginationPosition === "top" || paginationPosition === "both") &&
        pagerRow}
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead
                  key={header.id}
                  className={cn(
                    header.column.columnDef.meta?.hiddenOnMobile &&
                      "hidden md:table-cell"
                  )}
                >
                  {header.isPlaceholder ? null : (
                    <div className="flex flex-col gap-1.5 py-1">
                      <DataTableColumnHeader column={header.column}>
                        <table.FlexRender header={header} />
                      </DataTableColumnHeader>
                      {filtersOpen && header.column.columnDef.meta?.filterVariant && (
                        <div className="hidden sm:block">
                          <DataTableFilterInput column={header.column} />
                        </div>
                      )}
                    </div>
                  )}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={leafColumnCount}
                className="text-center text-sm text-muted-foreground"
              >
                {t.dataTableNoResults}
              </TableCell>
            </TableRow>
          ) : groupBy ? (
            groupBy.order.map((sectionValue) => {
              const sectionRows = rows.filter(
                (row) => row.original[groupBy.key] === sectionValue
              )
              if (sectionRows.length === 0) return null
              return (
                <React.Fragment key={String(sectionValue)}>
                  <TableRow className="hover:bg-transparent">
                    <TableCell
                      colSpan={leafColumnCount}
                      className="bg-muted/50 text-xs font-medium text-muted-foreground"
                    >
                      {groupBy.label ? groupBy.label(sectionValue) : String(sectionValue)}
                    </TableCell>
                  </TableRow>
                  {sectionRows.map(renderDataRow)}
                </React.Fragment>
              )
            })
          ) : (
            rows.map(renderDataRow)
          )}
        </TableBody>
      </Table>
      {!groupBy &&
        (paginationPosition === "bottom" || paginationPosition === "both") &&
        pagerRow}
    </div>
  )
}
