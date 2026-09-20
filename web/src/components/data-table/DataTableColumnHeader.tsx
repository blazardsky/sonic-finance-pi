import { RiArrowDownSLine, RiArrowUpDownLine, RiArrowUpSLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
import type { Column, RowData } from "@tanstack/react-table"
import type { DataTableFeatures } from "./features"

// Wraps a column's already-rendered label with a sort toggle when the column
// is sortable — a plain span otherwise (an Actions or expand-toggle column
// never sorts). Two-state cycle (asc/desc, never back to unsorted) matching
// shadcn's own data-table recipe: once a household starts sorting a column,
// clicking again flips direction rather than clearing it.
export function DataTableColumnHeader<TData extends RowData>({
  column,
  children,
}: {
  column: Column<DataTableFeatures, TData, unknown>
  children: React.ReactNode
}) {
  if (!column.getCanSort()) {
    return <span>{children}</span>
  }

  const sorted = column.getIsSorted()
  const Icon = sorted === "asc" ? RiArrowUpSLine : sorted === "desc" ? RiArrowDownSLine : RiArrowUpDownLine

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className="-ml-2 h-8 gap-1 px-2 font-medium"
      onClick={() => column.toggleSorting(sorted === "asc")}
    >
      {children}
      <Icon className="size-3.5 text-muted-foreground" />
    </Button>
  )
}
