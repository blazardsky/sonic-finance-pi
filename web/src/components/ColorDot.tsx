import { paletteVar, type ColorSlot } from "@/lib/palette"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

// A small filled circle in the slot's own primary — the swatch every table
// row and picker renders next to a name (spec, "Color usage"). `light-dark()`
// picks the right variant on its own, off the same color-scheme index.css
// already sets per theme, so no theme lookup is needed here. Shared by
// Categories.tsx's management tables and every Category/Subcategory
// `<SelectItem>` (Expenses/Incomes/RecurringExpenses), so the same dot means
// the same thing everywhere a Category is named.
export function ColorDot({
  color,
  className,
}: {
  color: ColorSlot
  className?: string
}) {
  return (
    <span
      aria-hidden
      className={cn("inline-block size-3 shrink-0 rounded-full", className)}
      style={{ backgroundColor: paletteVar(color, "primary") }}
    />
  )
}

// A Category/Subcategory name as a badge in its own palette color (ticket
// 04), so the same grouping ColorDot signals next to a picker is visible at
// a glance in the Expense/Income lists too. Colors, not the badge component's
// own variants: those are fixed CSS classes, and this needs whichever of the
// 9 slots the Category actually carries.
export function CategoryBadge({
  name,
  color,
  className,
}: {
  name: string
  color: ColorSlot
  className?: string
}) {
  return (
    <Badge
      className={cn("border-transparent", className)}
      style={{
        backgroundColor: paletteVar(color, "background"),
        color: paletteVar(color, "foreground"),
      }}
    >
      {name}
    </Badge>
  )
}
