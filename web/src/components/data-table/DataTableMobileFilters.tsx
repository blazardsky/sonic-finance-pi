import * as React from "react"
import { RiFilter3Line } from "@remixicon/react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet"
import { t } from "@/lib/strings"
import type { ReactTable, RowData } from "@tanstack/react-table"
import { DataTableFilterInput } from "./DataTableFilterInput"
import type { DataTableFeatures } from "./features"

// The mobile counterpart to the desktop inline header filters — there is no
// column header to attach a filter to on a narrow screen, so this reads and
// writes the exact same `columnFilters` state via a bottom sheet instead.
// Renders nothing when the table has no filterable column.
export function DataTableMobileFilters<TData extends RowData>({
  table,
}: {
  table: ReactTable<DataTableFeatures, TData>
}) {
  const [open, setOpen] = React.useState(false)
  const filterableHeaders = table
    .getHeaderGroups()
    .flatMap((group) => group.headers)
    .filter((header) => header.column.columnDef.meta?.filterVariant)

  if (filterableHeaders.length === 0) return null

  const activeCount = filterableHeaders.filter(
    (header) => header.column.getFilterValue() !== undefined
  ).length

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="w-fit gap-1.5 sm:hidden"
        onClick={() => setOpen(true)}
      >
        <RiFilter3Line className="size-4" />
        {t.dataTableFilters}
        {activeCount > 0 && (
          <Badge variant="secondary" className="px-1.5">
            {activeCount}
          </Badge>
        )}
      </Button>
      <SheetContent side="bottom" className="max-h-[80vh] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>{t.dataTableFilters}</SheetTitle>
        </SheetHeader>
        <div className="flex flex-col gap-4 px-4 pb-4">
          {filterableHeaders.map((header) => (
            <div key={header.id} className="flex flex-col gap-1.5">
              <span className="text-sm font-medium">
                <table.FlexRender header={header} />
              </span>
              <DataTableFilterInput column={header.column} />
            </div>
          ))}
          {activeCount > 0 && (
            <Button
              type="button"
              variant="ghost"
              onClick={() =>
                filterableHeaders.forEach((header) =>
                  header.column.setFilterValue(undefined)
                )
              }
            >
              {t.dataTableClearFilters}
            </Button>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}
