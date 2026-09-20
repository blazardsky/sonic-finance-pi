import { RiArrowLeftSLine, RiArrowRightSLine } from "@remixicon/react"

import { Button } from "@/components/ui/button"
import { t } from "@/lib/strings"
import type { RowData, Table } from "@tanstack/react-table"
import type { DataTableFeatures } from "./features"

// Extracted from the arrows Expenses.tsx/Incomes.tsx used to copy-paste
// between themselves (icon-only below `sm`, text+icon above) — every
// migrated page gets this, not just those two, now driven by the table
// instance's own pagination state instead of manual page/limit/offset.
export function DataTablePagination<TData extends RowData>({
  table,
}: {
  table: Table<DataTableFeatures, TData>
}) {
  return (
    <div className="flex items-center gap-2">
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-label={t.previousPage}
        disabled={!table.getCanPreviousPage()}
        onClick={() => table.previousPage()}
      >
        <RiArrowLeftSLine className="sm:hidden" />
        <span className="hidden sm:inline">{t.previousPage}</span>
      </Button>
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-label={t.nextPage}
        disabled={!table.getCanNextPage()}
        onClick={() => table.nextPage()}
      >
        <RiArrowRightSLine className="sm:hidden" />
        <span className="hidden sm:inline">{t.nextPage}</span>
      </Button>
    </div>
  )
}
