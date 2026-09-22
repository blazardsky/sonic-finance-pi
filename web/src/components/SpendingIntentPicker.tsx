import { RiQuestionLine } from "@remixicon/react"

import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
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

// One badge in a pair/trio: a plain click-to-toggle button, not a native
// radio — clicking it selects it (and its onClick decides what happens to
// its pair-partner(s)). The `default`/`outline` variant swap is the "dims"
// half of the spec's "selecting one dims/disables its pair-partner": the
// unselected side of a group always reads as the plainer, less prominent
// badge. Sized for a thumb (h-10, generous horizontal padding) and grown to
// fill its row with `flex-1` on the wrapping button, rather than the compact
// chip Badge renders everywhere else it's used — this is a tap target first,
// a label second.
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
    <button type="button" role="radio" aria-checked={selected} onClick={onClick} className="flex-1">
      <Badge
        variant={selected ? "default" : "outline"}
        className={cn("h-10 w-full justify-center px-3 text-sm", !selected && "text-muted-foreground")}
      >
        {label}
      </Badge>
    </button>
  )
}

// The "?" beside every Spending intent label (picker and section heading
// alike) — the one place the values are explained, so the picker itself
// never needs hint text of its own competing for width with the badges.
export function SpendingIntentHelpTooltip() {
  return (
    <Tooltip>
      <TooltipTrigger type="button" className="text-muted-foreground">
        <RiQuestionLine className="size-4" />
      </TooltipTrigger>
      <TooltipContent className="max-w-64">{t.spendingIntentHelp}</TooltipContent>
    </Tooltip>
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
// null throughout. Re-tapping Necessità/Desiderio is how the top pair clears
// back to null — not a separate "none" control, since a click-to-toggle
// badge already doubles as its own undo. The second row is a true 3-way
// radio instead: Sensata/Neutro/Stronzata always has exactly one of the
// three selected once Desiderio is active (Neutro the moment Desiderio
// itself is picked, same plain "desire" value this already was before
// Neutro had its own badge) — nobody is forced to call a want either wise or
// a mistake just to record that it *was* a want.
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
      <div className="flex gap-2">
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
        <div className="flex gap-2">
          <IntentBadge
            label={t.spendingIntentWise}
            selected={value === "desire_wise"}
            onClick={() => onChange("desire_wise")}
          />
          <IntentBadge
            label={t.spendingIntentNeutral}
            selected={value === "desire"}
            onClick={() => onChange("desire")}
          />
          <IntentBadge
            label={t.spendingIntentBullshit}
            selected={value === "desire_bullshit"}
            onClick={() => onChange("desire_bullshit")}
          />
        </div>
      )}
    </div>
  )
}
