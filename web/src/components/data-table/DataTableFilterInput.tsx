import { DatePicker } from "@/components/date-picker"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { t } from "@/lib/strings"
import type { Column, RowData } from "@tanstack/react-table"
import type { DataTableFeatures } from "./features"

// Radix Select refuses an empty-string item value (same bargain every other
// Select sentinel in this codebase makes) — this is the "no filter" option.
const NO_FILTER = "__all__"

type NumberRange = [number | undefined, number | undefined]
type DateRange = [string | undefined, string | undefined]

// Collapses a fully-cleared [undefined, undefined] tuple down to plain
// undefined, so getFilterValue() !== undefined (what the mobile sheet's
// active-filter count, and TanStack's own "has a filter" checks, both read)
// isn't left thinking a range/date-range column is still filtering on
// nothing.
function normalizeRange<T>(range: [T | undefined, T | undefined]): [T | undefined, T | undefined] | undefined {
  return range[0] === undefined && range[1] === undefined ? undefined : range
}

// Renders the widget a column's `meta.filterVariant` asks for, reading and
// writing that column's own filter value — used identically by the desktop
// inline header (DataTable.tsx) and the mobile filter sheet (ticket 04),
// which is what keeps the two in sync without a separate filter state.
export function DataTableFilterInput<TData extends RowData>({
  column,
}: {
  column: Column<DataTableFeatures, TData, unknown>
}) {
  const variant = column.columnDef.meta?.filterVariant
  const value = column.getFilterValue()

  if (variant === "text") {
    return (
      <Input
        value={(value as string | undefined) ?? ""}
        onChange={(e) => column.setFilterValue(e.target.value || undefined)}
        className="h-8"
      />
    )
  }

  if (variant === "select") {
    const options = Array.from(column.getFacetedUniqueValues().keys())
      .filter((v): v is string => typeof v === "string" && v !== "")
      .sort((a, b) => a.localeCompare(b))

    return (
      <Select
        value={(value as string | undefined) ?? NO_FILTER}
        onValueChange={(v) => column.setFilterValue(v === NO_FILTER ? undefined : v)}
      >
        <SelectTrigger className="h-8 w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NO_FILTER}>{t.dataTableFilterAll}</SelectItem>
          {options.map((opt) => (
            <SelectItem key={opt} value={opt}>
              {opt}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    )
  }

  if (variant === "range") {
    const [min, max] = (value as NumberRange | undefined) ?? [undefined, undefined]
    return (
      <div className="flex gap-1">
        <Input
          type="number"
          inputMode="decimal"
          placeholder={t.dataTableFilterMin}
          value={min ?? ""}
          onChange={(e) =>
            column.setFilterValue((old: NumberRange | undefined) =>
              normalizeRange<number>([
                e.target.value === "" ? undefined : Number(e.target.value),
                old?.[1],
              ])
            )
          }
          className="h-8 w-1/2"
        />
        <Input
          type="number"
          inputMode="decimal"
          placeholder={t.dataTableFilterMax}
          value={max ?? ""}
          onChange={(e) =>
            column.setFilterValue((old: NumberRange | undefined) =>
              normalizeRange<number>([
                old?.[0],
                e.target.value === "" ? undefined : Number(e.target.value),
              ])
            )
          }
          className="h-8 w-1/2"
        />
      </div>
    )
  }

  if (variant === "date-range") {
    const [from, to] = (value as DateRange | undefined) ?? [undefined, undefined]
    return (
      <div className="flex gap-1">
        <DatePicker
          value={from ?? ""}
          onValueChange={(v) =>
            column.setFilterValue((old: DateRange | undefined) =>
              normalizeRange<string>([v || undefined, old?.[1]])
            )
          }
          placeholder={t.dataTableFilterMin}
          className="h-8 w-1/2 px-2 text-xs"
        />
        <DatePicker
          value={to ?? ""}
          onValueChange={(v) =>
            column.setFilterValue((old: DateRange | undefined) =>
              normalizeRange<string>([old?.[0], v || undefined])
            )
          }
          placeholder={t.dataTableFilterMax}
          className="h-8 w-1/2 px-2 text-xs"
        />
      </div>
    )
  }

  return null
}
