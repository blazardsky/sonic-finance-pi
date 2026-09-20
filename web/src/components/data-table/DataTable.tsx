/* eslint-disable react-refresh/only-export-components */
import * as React from "react"
import { RiArrowDownSLine, RiArrowRightSLine } from "@remixicon/react"
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
} from "@tanstack/react-table"

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
import { dataTableFeatures, type DataTableFeatures } from "./features"
import { DataTableColumnHeader } from "./DataTableColumnHeader"
import { DataTableFilterInput } from "./DataTableFilterInput"
import { DataTableMobileFilters } from "./DataTableMobileFilters"
import { DataTablePagination } from "./DataTablePagination"

// Every column ticket 01 defines is a data column; this one is injected by
// DataTable itself when `renderSubRow` is set, always first, never supplied
// by a page.
const EXPAND_COLUMN_ID = "__expand__"

export type DataTableColumnDef<TData extends RowData> = ColumnDef<
  DataTableFeatures,
  TData
>

// A page defines its columns against this rather than raw createColumnHelper,
// so every migrated page gets the same feature set without repeating the
// generic — `createDataTableColumnHelper<Expense>()` in place of
// `createColumnHelper<typeof dataTableFeatures, Expense>()`.
export function createDataTableColumnHelper<TData extends RowData>() {
  return createColumnHelper<DataTableFeatures, TData>()
}

const DEFAULT_PAGE_SIZE = 10

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
}: {
  columns: DataTableColumnDef<TData>[]
  data: TData[]
  initialSorting?: SortingState
  pageSize?: number
  getRowId?: (row: TData, index: number) => string
  paginationPosition?: "top" | "bottom" | "both"
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
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize,
  })
  const [expanded, setExpanded] = React.useState<ExpandedState>({})

  const tableColumns = React.useMemo(() => {
    if (!renderSubRow) return columns
    const expandColumn: DataTableColumnDef<TData> = {
      id: EXPAND_COLUMN_ID,
      header: () => null,
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
    return [expandColumn, ...columns]
  }, [columns, renderSubRow])

  const table = useTable({
    features: dataTableFeatures,
    columns: tableColumns,
    data,
    getRowId,
    getRowCanExpand: renderSubRow ? () => true : undefined,
    state: { sorting, columnFilters, pagination, expanded },
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

  const renderDataRow = (row: Row<DataTableFeatures, TData>) => (
    <React.Fragment key={row.id}>
      <TableRow>
        {row.getAllCells().map((cell) => (
          <TableCell key={cell.id}>
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
      {!groupBy &&
        (paginationPosition === "top" || paginationPosition === "both") &&
        pager}
      <Table>
        <TableHeader>
          {table.getHeaderGroups().map((headerGroup) => (
            <TableRow key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <TableHead key={header.id}>
                  {header.isPlaceholder ? null : (
                    <div className="flex flex-col gap-1.5 py-1">
                      <DataTableColumnHeader column={header.column}>
                        <table.FlexRender header={header} />
                      </DataTableColumnHeader>
                      {header.column.columnDef.meta?.filterVariant && (
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
        pager}
    </div>
  )
}
