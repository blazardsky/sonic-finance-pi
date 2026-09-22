import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { t } from "@/lib/strings"
import type { SpendingIntent } from "@/types"

// The Necessity/Desire pair is mutually exclusive with itself, and Wise/
// Bullshit only ever refines a Desire — so "is anything under Desire active"
// is the one question both the pair's own selected state and the Wise/
// Bullshit pair's visibility depend on.
function isDesireish(value: SpendingIntent | null): boolean {
  return value === "desire" || value === "desire_wise" || value === "desire_bullshit"
}

// One badge in a pair: a plain click-to-toggle button, not a native radio —
// clicking it selects it (and its onClick decides what happens to its
// pair-partner), clicking it again while selected is how the pair clears
// back to nothing. The `default`/`outline` variant swap is the "dims" half
// of the spec's "selecting one dims/disables its pair-partner": the unselected
// side of a pair always reads as the plainer, less prominent badge.
function IntentBadge({
  label,
  selected,
  onClick,
}: {
  label: string
  selected: boolean
  onClick: () => void
}) {
  return (
    <button type="button" role="radio" aria-checked={selected} onClick={onClick}>
      <Badge variant={selected ? "default" : "outline"} className={cn(!selected && "text-muted-foreground")}>
        {label}
      </Badge>
    </button>
  )
}

// The Spending intent section's two-pair badge control (spec's Implementation
// Decisions → Frontend): Necessity/Desire, then — only once Desire is picked,
// in any of its three shades — Wise/Bullshit appears beneath it to refine it.
// Shared by the Category edit form (ticket 02), the Expense form and the
// Recurring expense form (tickets 03-04) alike, so all three read the exact
// same interaction.
//
// The whole thing is optional (spec, story 14): value is SpendingIntent |
// null throughout, and re-tapping whichever badge is currently selected in a
// pair is how it clears back to null (Necessity/Desire pair) or back to plain
// "desire" (Wise/Bullshit pair) — not a separate "none" control, since a
// click-to-toggle badge already doubles as its own undo.
export function SpendingIntentPicker({
  value,
  onChange,
}: {
  value: SpendingIntent | null
  onChange: (value: SpendingIntent | null) => void
}) {
  const desireish = isDesireish(value)

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-2">
        <IntentBadge
          label={t.spendingIntentNecessity}
          selected={value === "necessity"}
          onClick={() => onChange(value === "necessity" ? null : "necessity")}
        />
        <IntentBadge
          label={t.spendingIntentDesire}
          selected={desireish}
          onClick={() => onChange(desireish ? null : "desire")}
        />
      </div>
      {desireish && (
        <div className="flex flex-wrap gap-2">
          <IntentBadge
            label={t.spendingIntentWise}
            selected={value === "desire_wise"}
            onClick={() => onChange(value === "desire_wise" ? "desire" : "desire_wise")}
          />
          <IntentBadge
            label={t.spendingIntentBullshit}
            selected={value === "desire_bullshit"}
            onClick={() => onChange(value === "desire_bullshit" ? "desire" : "desire_bullshit")}
          />
        </div>
      )}
    </div>
  )
}
