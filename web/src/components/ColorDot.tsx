import { paletteVar, type ColorSlot } from "@/lib/palette"
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
